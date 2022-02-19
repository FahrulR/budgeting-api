#!/bin/sh

for file in `find $1 | grep -i '.sql' | sort -n`
do
  echo "Applying $file"
  echo "DB str: $DB_CONNECTION_STRING_MIGRATE"
  docker exec -ti ${COMPONENT}_database_1 psql -f $file $DB_CONNECTION_STRING_MIGRATE
done