import asyncio
from concurrent.futures import ProcessPoolExecutor, ThreadPoolExecutor, as_completed
import configparser
import datetime
import gzip
import logging
import os
import sys
from pyshark.capture.capture import TSharkCrashException
import pyshark
from os.path import basename

logger=logging.getLogger(__name__)
config=configparser.ConfigParser()
config.read("config.ini")
def pcapInfoToListBy2Filters(filename, filter1, filter2, eventloop=None):
    csvlist=[]
    display_filter=filter1+"||"+filter2
    field1=filter1.split('.')[-1].lower()
    field2=filter2.split('.')[-1].lower()
    try:
        with pyshark.FileCapture(filename,only_summaries=True,display_filter=display_filter,eventloop=eventloop) as capsummary:
            try:
                with pyshark.FileCapture(filename,only_summaries=False,display_filter=display_filter,use_ek=True,eventloop=eventloop) as capdetail: 
                    #capdetail.load_packets()
                    #capsummary.load_packets()
                    for packetsummary, packetdetail in zip(capsummary, capdetail): 
                            date=packetdetail.sniff_timestamp if config["python"]["timestamptodate"]=="False" else datetime.datetime.fromtimestamp(float(packetdetail.sniff_timestamp))
                            csvlist.append([basename(filename),packetsummary.no,date,
                                            packetsummary.source,packetsummary.destination,
                                            packetsummary.protocol,packetsummary.info,
                                            packetdetail.s1ap.get_field(field1.upper()),
                                            packetdetail.s1ap.get_field(field2.upper())])
                    logger.debug(f"length of summary capture : {len(capsummary)}, length of detail capture : {len(capdetail)}")
            except TSharkCrashException:
                print("detail cap catched")
                logger.debug(f"filename:{filename} : detail cap catched")
    except TSharkCrashException:
            print(f"filename:{filename} : summary cap catched")
    print("print \"%s\" successful"%filename)
    logger.debug(f"filename:{filename} : successful")
    return csvlist



def process_one_file_by2filters(csvwriter,filename,filter1,filter2):
    csvwriter.writerows(pcapInfoToListBy2Filters(filename,filter1,filter2))
    
def sctpanalysis(csvfile_id,csvwriter_id, sctp_file_list,cache_path,filter1,filter2, mode=0):
    logger.info("sctp started")
    csvwriter_id.writerow(
        [
            "Filename",
            "Pkt Num",
            "Time",
            "Source IP",
            "Destination IP",
            "Protocol",
            "Summary Info",
            "MME-ID",
            "ENB-ID",
        ]
    ) 
    for i, filename in enumerate(sctp_file_list):
        if filename[-3:] == ".gz":
            with gzip.open(filename) as f:
                with open(
                    os.path.join(
                        cache_path, os.path.splitext(os.path.basename(filename))[0]
                    ),
                    "wb",
                ) as f2:
                    f2.write(f.read())
                    sctp_file_list[i] = f2.name
    if mode == 0:
        for filename in sctp_file_list:
            # csvwriter_id.writerow([os.path.basename(filename),'','','','','','',''])
            process_one_file_by2filters(csvwriter_id, filename, filter1, filter2)
            csvfile_id.flush()
            print("sctp_finished_one")
            sys.stdout.flush()
    elif mode == 1:
        with ThreadPoolExecutor() as executor:
            fs = [
                executor.submit(
                    pcapInfoToListBy2Filters,
                    filename,
                    filter1,
                    filter2,
                    asyncio.new_event_loop(),
                )
                for filename in sctp_file_list
            ]
            for future in as_completed(fs):
                csvwriter_id.writerows(future.result())
                csvfile_id.flush()
                print("sctp_finished_one")
                sys.stdout.flush()
        print("multithread success")
    elif mode == 2:
        with ProcessPoolExecutor() as executor:
            fs = [
                executor.submit(
                    pcapInfoToListBy2Filters,
                    filename,
                    filter1,
                    filter2,
                )
                for filename in sctp_file_list
            ]
            for future in as_completed(fs):
                csvwriter_id.writerows(future.result())
                csvfile_id.flush()
                print("sctp_finished_one")
                sys.stdout.flush()
        print("multithread success")
    logger.info("sctp finished")