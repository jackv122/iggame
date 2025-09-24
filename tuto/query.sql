-- 1) User history for August 2025 (partition-prunable, uses idx_user_time)
EXPLAIN PARTITIONS
SELECT id, gameid, roundid, payout, updatetime
FROM betting
WHERE userid = 42
  AND updatetime >= '2025-08-01'
  AND updatetime <  '2025-09-01'
ORDER BY updatetime, id
LIMIT 100;

-- 2) Game stats in a day (partition-prunable, uses idx_game_time)
SELECT gameid, COUNT(*) AS bets, SUM(payout) AS total_payout
FROM betting
WHERE gameid = '001'
  AND updatetime >= '2025-08-15'
  AND updatetime <  '2025-08-16'
GROUP BY gameid;

-- 3) Count in a month (partition-prunable)
SELECT COUNT(*) AS cnt
FROM betting
WHERE updatetime >= '2025-08-01'
  AND updatetime <  '2025-09-01';

-- 4) Keyset pagination (stable, uses (userid, updatetime, id))
-- first page
SELECT id, updatetime, payout
FROM betting
WHERE userid = 42
  AND updatetime >= '2025-08-01'
  AND updatetime <  '2025-09-01'
ORDER BY updatetime, id
LIMIT 100;

-- next page (use last (updatetime, id) from previous page)
SELECT id, updatetime, payout
FROM betting
WHERE userid = 42
  AND updatetime >= '2025-08-01'
  AND updatetime <  '2025-09-01'
  AND (updatetime, id) > ('2025-08-03 12:34:56', 123456789)
ORDER BY updatetime, id
LIMIT 100;