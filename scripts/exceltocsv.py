import pandas as pd
import os 
def tocsv(filename):
    xls = pd.ExcelFile(filename)
    all_data=pd.DataFrame()
    for sheet_name in xls.sheet_names:
        df=pd.read_excel(xls,sheet_name=sheet_name,usecols=[0])
        df.columns=['Event Name']
        df['Category']=sheet_name
        all_data=pd.concat([all_data,df],ignore_index=True)
    all_data.to_csv(os.path.splitext(filename)[0]+".csv",index=False)
    
if __name__ == '__main__':
    tocsv('./dbg信令分类_唯一分类.xlsx')