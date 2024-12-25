# for 3 to 16, loop through the weeks and echo the week number
for i in {1..16}
do
  echo "Week $i"
  curl "http://nfl.mcgizzle.casa/backfill?week=$i&season=2024"
  echo "Finished Week $i"
done
