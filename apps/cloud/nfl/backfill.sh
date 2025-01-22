echo "Season Type $2"
for i in {1..16}
do
  echo "Week $i"
  curl "http://nfl.mcgizzle.casa/backfill?week=$i&season=2024&seasontype=2"
  echo "Finished Week $i"
done
