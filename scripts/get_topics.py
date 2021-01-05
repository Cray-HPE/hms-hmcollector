#!/usr/bin/env python3

#  Copyright 2020 Hewlett Packard Enterprise Development LP

import time
from kafka import KafkaConsumer
from kafka import errors


def get_kafka_topics(port):
    broker_list = "cluster-kafka-bootstrap.sma.svc.cluster.local:%d" % port
    print(broker_list)
    topic_list = {}
    retry = 1
    while True:
        try:
            consumer = KafkaConsumer(group_id='test', bootstrap_servers=broker_list)
        except errors.NoBrokersAvailable:
            print("NoBrokersAvailable Exception (%d)" % retry)
            if retry >= 3:
                raise
            else:
                time.sleep(1)
                retry += 1
        except Exception as e:
            print("KafkaConsumer Exception")
            raise
        else:
            for t in consumer.topics():
                if t.find('internal-', 0, 9) != 0:  # strip out hidden topics
                    topic_list[t] = len(consumer.partitions_for_topic(t))
            break

    return topic_list


topics = get_kafka_topics(9092)
print(topics)
