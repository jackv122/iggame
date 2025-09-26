-- STORE PARTITION PROCEDURE --------------------------------
DELIMITER //
CREATE PROCEDURE partition_betting_monthly(IN p_start DATE, IN p_end DATE)
BEGIN
  SET @ddl = 'ALTER TABLE betting PARTITION BY RANGE COLUMNS(updatetime) (';
  SET @d = p_start;
  WHILE @d < p_end DO
    SET @next = DATE_ADD(@d, INTERVAL 1 MONTH);
    SET @pname = DATE_FORMAT(@d, 'p%Y_%m');
    SET @ddl = CONCAT(
      @ddl,
      'PARTITION ', @pname, ' VALUES LESS THAN (''',
      DATE_FORMAT(@next, '%Y-%m-%d'),
      '''),'
    );
    SET @d = @next;
  END WHILE;
  -- Remove trailing comma and close parentheses
  SET @ddl = CONCAT(TRIM(TRAILING ',' FROM @ddl), ')');
  PREPARE s FROM @ddl; EXECUTE s; DEALLOCATE PREPARE s;
END//
DELIMITER ;
-- Example: partition from 2025-01 to 2027-01
CALL partition_betting_monthly('2025-01-01','2027-01-01');


-- STORE DROP PARTITIONS PROCEDURE --------------------------------
DELIMITER //
CREATE PROCEDURE drop_betting_partitions_before(IN p_cutoff DATE)
BEGIN
  -- STEP 1: Show all partitions for table `betting`
  SELECT partition_name, partition_description
  FROM information_schema.PARTITIONS
  WHERE table_schema = DATABASE()
    AND table_name = 'betting'
  ORDER BY partition_ordinal_position;

  -- (boundary strictly before the first day of cutoff month, exclude MAXVALUE)
  SET @cut := DATE_FORMAT(p_cutoff, '%Y-%m-01');
  SELECT 
    partition_name,
    partition_description,
    CONCAT('ALTER TABLE `betting` DROP PARTITION ', partition_name) AS would_drop_sql
  FROM information_schema.PARTITIONS
  WHERE table_schema = DATABASE()
    AND table_name = 'betting'
    AND partition_name IS NOT NULL
    AND partition_description IS NOT NULL
    AND STR_TO_DATE(REPLACE(partition_description, '''', ''), '%Y-%m-%d') <= @cut
  ORDER BY partition_ordinal_position;

  -- STEP 3: Actually drop the partitions (<= cutoff)
  BEGIN
    DECLARE done INT DEFAULT 0;
    DECLARE p_name VARCHAR(64);
    DECLARE cur CURSOR FOR
      SELECT partition_name
      FROM information_schema.PARTITIONS
      WHERE table_schema = DATABASE()
        AND table_name = 'betting'
        AND partition_name IS NOT NULL
        AND partition_description IS NOT NULL
        AND STR_TO_DATE(REPLACE(partition_description, '''', ''), '%Y-%m-%d') <= @cut
      ORDER BY partition_ordinal_position;
    DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = 1;

    OPEN cur;
    drop_loop: LOOP
      FETCH cur INTO p_name;
      IF done THEN LEAVE drop_loop; END IF;
      SET @sql = CONCAT('ALTER TABLE `betting` DROP PARTITION ', p_name);
      SELECT @sql AS exec_sql;  -- log
      PREPARE s FROM @sql; EXECUTE s; DEALLOCATE PREPARE s;
    END LOOP;
    CLOSE cur;
  END;
END//
DELIMITER ;

-- Drops all partitions strictly before 2025-01 (i.e., < '2025-01-01')
CALL drop_betting_partitions_before('2025-09-01');

DELIMITER //
CREATE PROCEDURE add_betting_partitions_monthly(IN p_start DATE, IN p_end DATE)
BEGIN
  DECLARE d DATE;
  DECLARE next_m DATE;
  DECLARE pname VARCHAR(16);
  DECLARE cnt INT;

  SET d = DATE_FORMAT(p_start, '%Y-%m-01');
  WHILE d < p_end DO
    SET next_m = DATE_ADD(d, INTERVAL 1 MONTH);
    SET pname = DATE_FORMAT(d, 'p%Y_%m');

    -- Skip if this monthly partition already exists
    SELECT COUNT(*) INTO cnt
    FROM information_schema.PARTITIONS
    WHERE table_schema = DATABASE()
      AND table_name = 'betting'
      AND partition_name = pname;

    IF cnt = 0 THEN
      SET @sql = CONCAT(
        'ALTER TABLE betting ADD PARTITION (',
        'PARTITION ', pname, ' VALUES LESS THAN (''', DATE_FORMAT(next_m, '%Y-%m-%d'), '''))'
      );
      SELECT @sql AS exec_sql; -- log
      PREPARE s FROM @sql; EXECUTE s; DEALLOCATE PREPARE s;
    END IF;

    SET d = next_m;
  END WHILE;
END//
DELIMITER ;


-- Example: create monthly partitions for 2025
-- CALL add_betting_partitions_monthly('2025-01-01','2026-01-01');
