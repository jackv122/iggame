

mysql --host=localhost --port=3306 --user=root --password=hailuava12a6 vwallet_001

ALTER TABLE betting
  MODIFY COLUMN updatetime DATETIME NOT NULL
  DEFAULT CURRENT_TIMESTAMP
  ON UPDATE CURRENT_TIMESTAMP;

-- Adjust PK to include the partition key
ALTER TABLE betting
  DROP PRIMARY KEY,
  ADD PRIMARY KEY (id, updatetime),
  DROP INDEX createtime,
  ADD INDEX idx_user_time (userid, updatetime, id),
  ADD INDEX idx_game_time (gameid, updatetime, id);

INSERT INTO betting
  (gameid, gamenumber, roomid, userid, roundid, betdetail, result, payout, payedout, h)
VALUES
  ('001', 123456789, '000001', 42, 1, '{"bets":[{"type":"F7","amt":1.5}]}', '7', 0, 0, 'txhash_or_sig_here');

CREATE TABLE `betting` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `gameid` varchar(3) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `gamenumber` bigint NOT NULL,
  `roomid` varchar(6) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `userid` int NOT NULL,
  `roundid` int NOT NULL,
  `betdetail` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `result` varchar(45) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `payout` double NOT NULL DEFAULT '0',
  `payedout` tinyint NOT NULL DEFAULT '0',
  `updatetime` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `h` varchar(44) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`,`updatetime`),
  KEY `gameId` (`gameid`),
  KEY `roomId` (`roomid`),
  KEY `userid` (`userid`),
  KEY `gamenumber` (`gamenumber`),
  KEY `idx_user_time` (`userid`,`updatetime`,`id`),
  KEY `idx_game_time` (`gameid`,`updatetime`,`id`)
) ENGINE=InnoDB AUTO_INCREMENT=3136 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;



-- Enable partitioning by months


--EX:
    ALTER TABLE betting PARTITION BY RANGE COLUMNS(updatetime) (
        PARTITION p2025_01 VALUES LESS THAN ('2025-02-01'),
        PARTITION p2025_02 VALUES LESS THAN ('2025-03-01'),
        PARTITION p2025_03 VALUES LESS THAN ('2025-04-01'),
        PARTITION pmax VALUES LESS THAN (MAXVALUE)
    );



-- Add next month --------------------------------
SET @first_day_next := DATE_FORMAT(DATE_ADD(CURDATE(), INTERVAL 1 MONTH), '%Y-%m-01');
SET @pname := DATE_FORMAT(DATE_ADD(CURDATE(), INTERVAL 0 MONTH), 'p%Y_%m');
SET @sql := CONCAT(
  'ALTER TABLE betting ADD PARTITION (PARTITION ', @pname,
  ' VALUES LESS THAN (''', @first_day_next, '''))'
);
PREPARE s FROM @sql; EXECUTE s; DEALLOCATE PREPARE s;

-- EX: 
    ALTER TABLE betting
    ADD PARTITION (
    PARTITION p2025_10 VALUES LESS THAN ('2025-11-01')
    );

-- drop old month --------------------------------
SET @old := DATE_FORMAT(DATE_SUB(CURDATE(), INTERVAL 12 MONTH), 'p%Y_%m');
SET @sql := CONCAT('ALTER TABLE betting DROP PARTITION ', @old);
-- You may want to check information_schema.PARTITIONS first
PREPARE s FROM @sql; EXECUTE s; DEALLOCATE PREPARE s;
-- EX:
    ALTER TABLE betting DROP PARTITION p2025_01;

    ALTER TABLE betting DROP PARTITION (
        p2025_01,
        p2025_02
    );


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
  SET @ddl = CONCAT(@ddl, 'PARTITION pmax VALUES LESS THAN (MAXVALUE))');
  PREPARE s FROM @ddl; EXECUTE s; DEALLOCATE PREPARE s;
END//
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
    AND partition_description <> 'MAXVALUE'
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
        AND partition_description <> 'MAXVALUE'
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
