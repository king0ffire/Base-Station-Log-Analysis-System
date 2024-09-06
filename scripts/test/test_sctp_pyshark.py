from scapy.all import rdpcap
import csv
import pyshark
import gzip
import glob
import os
from datetime import datetime

debug=False
mmeposincsv=6
enbposincsv=7
log_name="Log_20240618_092153"
base_path=r"E:/"+log_name+"/logs"
file_list=glob.glob(base_path+r"\sctp*")

csvfile=open(os.path.join(base_path,log_name)+".csv","w",newline='')
csvwriter=csv.writer(csvfile)
for filename in file_list:
    
    if False: #(filename[-3:]=='.gz'):
        with gzip.open(filename) as f:
            with open(os.path.splitext(filename)[0],'wb') as f2:
                f2.write(f.read())
        capsummary = pyshark.FileCapture(os.path.splitext(filename)[0],only_summaries=True,display_filter='s1ap')
        capsummary.load_packets()
        '''
        filterCounter=0
        with gzip.open(filename) as f:
            pkts=rdpcap(f)
            datetimelist=[datetime.fromtimestamp(float(pkt.time)) for pkt in pkts]
            
            for pkt in pkts:
                print(pkt.show())
                if 's1ap' in pkt:
                    info=capsummary[filterCounter]
                    filterCounter+=1
                    pkttime=pkt.time
                    pktsrcip=pkt['ip'].src
                    pktdstip=pkt['ip'].dst
                    csvwriter.writerow([pkttime,pktsrcip,pktdstip,info])
        '''
        capdetailed=pyshark.FileCapture(os.path.splitext(filename)[0],only_summaries=False)
        capdetailedmmeid = pyshark.FileCapture(os.path.splitext(filename)[0],only_summaries=False,display_filter='s1ap.MME_UE_S1AP_ID')
        capdetailedenbid = pyshark.FileCapture(os.path.splitext(filename)[0],only_summaries=False,display_filter='s1ap.ENB_UE_S1AP_ID')
        #test=capdetailedmmeid[0]
        timeoffirstpacket=datetime.timestamp(capdetailed[0].sniff_time)
        idtosummarymap={}
        for i in range(len(capsummary)):
            #if 'S1AP' in capsummary[i]._fields['Protocol']:
                relativetime=capsummary[i].time
                calculatedtime=float(relativetime)+timeoffirstpacket
                srcip=capsummary[i]._fields['Source']
                dstip=capsummary[i]._fields['Destination']
                protocol=capsummary[i]._fields['Protocol']
                info=capsummary[i].info
                timeindate=datetime.fromtimestamp(calculatedtime)
                if debug:
                    originaldetailedtime=capdetailed[i].sniff_time
                    #if originaldetailedtime!=calculatedtime:
                        #print("error:%s %s",calculatedtime,originaldetailedtime)
                idtosummarymap[int(capsummary[i].no)]=[int(capsummary[i].no),calculatedtime,srcip,dstip,protocol,info,'','']
        for mmepacket in capdetailedmmeid:
             number=int(mmepacket.number)
             mmeid=mmepacket.s1ap.get_field_value('mme-ue-s1ap-id')
             idtosummarymap[number][mmeposincsv]=mmeid
        for enbpacket in capdetailedenbid:
             number=int(enbpacket.number)
             enbid=enbpacket.s1ap.get_field_value('enb-ue-s1ap-id')
        csvwriter.writerows(idtosummarymap.items())
    
    if 'sctp' in filename and filename[-3:]!='.gz':
            capsummary = pyshark.FileCapture(os.path.splitext(filename)[0],only_summaries=True,display_filter='s1ap')
            capsummary.load_packets()
            '''
            filterCounter=0
            with gzip.open(filename) as f:
                pkts=rdpcap(f)
                datetimelist=[datetime.fromtimestamp(float(pkt.time)) for pkt in pkts]
                
                for pkt in pkts:
                    print(pkt.show())
                    if 's1ap' in pkt:
                        info=capsummary[filterCounter]
                        filterCounter+=1
                        pkttime=pkt.time
                        pktsrcip=pkt['ip'].src
                        pktdstip=pkt['ip'].dst
                        csvwriter.writerow([pkttime,pktsrcip,pktdstip,info])
            '''
            capdetailed=pyshark.FileCapture(os.path.splitext(filename)[0],only_summaries=False)
            capdetailedmmeid = pyshark.FileCapture(os.path.splitext(filename)[0],only_summaries=False,display_filter='s1ap.MME_UE_S1AP_ID')
            capdetailedenbid = pyshark.FileCapture(os.path.splitext(filename)[0],only_summaries=False,display_filter='s1ap.ENB_UE_S1AP_ID')
            #test=capdetailedmmeid[0]
            timeoffirstpacket=datetime.timestamp(capdetailed[0].sniff_time)
            idtosummarymap={}
            for i in range(len(capsummary)):
                #if 'S1AP' in capsummary[i]._fields['Protocol']:
                    relativetime=capsummary[i].time
                    calculatedtime=float(relativetime)+timeoffirstpacket
                    srcip=capsummary[i]._fields['Source']
                    dstip=capsummary[i]._fields['Destination']
                    protocol=capsummary[i]._fields['Protocol']
                    info=capsummary[i].info
                    timeindate=datetime.fromtimestamp(calculatedtime)
                    if debug:
                        originaldetailedtime=capdetailed[i].sniff_time
                        #if originaldetailedtime!=calculatedtime:
                            #print("error:%s %s",calculatedtime,originaldetailedtime)
                    idtosummarymap[int(capsummary[i].no)]=[int(capsummary[i].no),calculatedtime,srcip,dstip,protocol,info,'','']
            for mmepacket in capdetailedmmeid:
                number=int(mmepacket.number)
                mmeid=mmepacket.s1ap.get_field_value('mme-ue-s1ap-id')
                idtosummarymap[number][mmeposincsv]=mmeid
            for enbpacket in capdetailedenbid:
                number=int(enbpacket.number)
                enbid=enbpacket.s1ap.get_field_value('enb-ue-s1ap-id')
            csvwriter.writerows(idtosummarymap.items())
                            

                        

            