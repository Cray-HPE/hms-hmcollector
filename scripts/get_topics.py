#!/usr/bin/env python3

# MIT License
#
# (C) Copyright [2020-2021] Hewlett Packard Enterprise Development LP
#
# Permission is hereby granted, free of charge, to any person obtaining a
# copy of this software and associated documentation files (the "Software"),
# to deal in the Software without restriction, including without limitation
# the rights to use, copy, modify, merge, publish, distribute, sublicense,
# and/or sell copies of the Software, and to permit persons to whom the
# Software is furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included
# in all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
# THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR
# OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
# ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
# OTHER DEALINGS IN THE SOFTWARE.

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
