# Thing Value History

## Objective

Provide historical reading of thing notifications and actions using the bucket store.

## Status

This service is reworked to fit into HiveKit as an independent module that can be incorporated anywhere in a module chain.

Limitations:

- Responses to requests are not recorded.
- Data capture uses simple filter rules on operation, thingID, and affordance name.
- Data retention using time based rules is out of scope.
- Averaging of historical data is out of scope.
- Storage monitoring is out of scope. This should be a bucketstore feature.
- This requires a backend storage that implements the IBucketStore interface.
- Analytics of historical information is out of scope. Recommended solution is to forward the captured data into an analytics backend.

The IBucketStore api is quite basic which is both a strength and weakness. The ability to add features by additional modules to operate on the stored data should be possible but is currently untested.

## Summary

The History module provides capture and retrieval of past notifications and actions. (responses are not tracked)

This operates as a pipe where request and notification messages flow through without impeding or modifying these messages.

Data ingress at a continuous rate of 1000 messages per second is readily supported on small systems with 500MB of RAM or more and plenty of disk space. Higher throughput can be supported if storage space, memory and CPU are available. Very-small environments with less than 100MB need some tuning of the amount of messages captured.

Basic data queries are provided through the API for the purpose to retrieve and compare historical information.

## for consideration

- use buckets that expire based on a time rule
  used for auto removal of low priority data
- rules for sample intervals
  used for storage of periodic data such as temp
- counters
  used for nr of motions/period

## Backend Storage

This service uses the bucket-store for the storage backend. The bucket-store supports several embedded backend implementations that run out of the box without the need for any setup and configuration.

Extending the bucket store with external databases such as Mongodb, SQLite, PostgresSQL and possibly others is under consideration. It is probably the best way forward for integration into analytics systems.

The bucket-store API provides a cursor with key-ranged seek capability which can be used for time-based queries. All bucket store implementations support this range query through cursors.

Currently, the Pebble bucket store is the default for the history store. It provides a good balance between storage size and resource usage for smaller systems. Pebble should be able to handle 1 TB of data or even more.

More testing is needed to determine the actual limitations and improve performance.

Note that the bucket store API is fairly basic. Ideally it is updated to support time queries better instead of relying the application to embed this into the keys.

## Performance

Performance is mostly limited by the messaging protocol used. The bench test shows an average read/write duration of 1-2 ms per 1000 calls which most backends can easily handle.

## Data Size

Data size of event samples depends on the type of sensor, actuator or service that captures the data. Below some example cases and the estimated memory to get an idea of the required space.

Since the store uses a bucket per thingID, the thingID itself does not add significant size. The key is the msec timestamp since epoc, approx 15 characters.

The following estimates are based on a sample size of 100 bytes uncompressed (key:20, event name:10, value: 60, json: 10). These are worst case numbers as deduplication and compression can reduce the sizes significantly.

Case 1: sensors with a 1 minute average sample interval.

- A single sensor -> 500K samples => 50MB/year (uncompressed)
- A small system with 10 sensors -> 5M samples => 500MB/year
- A medium system with 100 sensors -> 50M samples => 5GB/year
- A larger system with 1000 sensors -> 500M samples => 50GB/year

Case 2: sensors with a 1 second average sample interval.

- A single sensor -> 32M samples => 3.2GB/year (uncompressed)
- A small system with 10 sensors -> 320M samples => 32GB/year
- A larger system with 1000 sensors -> 32000 M samples => 3.2TB/year

In reality these numbers will be lower depending on the chosen store.

Case 3: image timelapse snapshot with 5 minute interval
An image is 720i compressed, around 100K/image.

- A single image -> 100K snapshots/year => 10 GB/year
- A system with 10 cameras -> 1000K snapshots/year => 100 GB/year
- A larger system with 100 cameras -> 10M snapshots/year => 1 TB/year

Backend Recommendations:

1. Use of kvstore backend is recommended for smallish datasets up to 100GB or so. Beyond this the read/write performance starts to suffer.
2. Of the embedded stores, Pebble scales best for large datasets of 100GB-10TB. Beyond that a stand-alone clustering database server should be used.
3. Bbolt works best when using a service that analyzes the data locally with heavy read operations. Write is slow but read is faster than the other stores when processing a lot of data.

### Retention - for consideration

Data that loses its meaningful usage after time can be removed or averaged using retention rules. The retention engine periodically removes records of events and actions from the store that meet the criteria. Rule criteria include the publishing agent, thingID and names.

## Configuration

This module is configured through a yaml file provided on startup. It defines a set of filter rules.

Q: does each thing have its own bucket?
Pro: easy to retire things and fast to query by thing
Con: can't use buckets for lifecyle management

```yaml
filters:
  notifications:
    # retain all events from thing1
    - messageType: event
      thingID: thing1
      names: [event1]
      retain: true
    # and ignore all property updates
    - messageType: property
      thingID: thing2
      retain: false

  requests:
    # retain all action requests
    - messageType: action
      retain: true

storage:
  - name: bucket-1
    filter: filter1

  - name: bucket-2
    filter: filter2
```
