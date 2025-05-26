# IMAGE SORTER

## Index a given directory so we know what's there

1. Scan the given directory
2. Hash each file in the directory
3. store this in a db (using the hash as the key)

## Print out list of duplicates

1. Run through the index (hashes stored in db)
2. Print out records where there are multiple locations

## Function to handle duplicates (which version to keep)

1. Iterate through files with multiple locations
2. Ask the user to decide which version we should keep

---

## Future improvements

* Only do this for image files

