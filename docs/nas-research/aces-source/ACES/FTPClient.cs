using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Net;
using System.IO;
using System.ComponentModel;
using System.Windows.Forms;
using System.Text.RegularExpressions;
using System.Net.Sockets;

namespace ATLNetwork.EDI.ACES
{
    public class FTPClient
    {
        private BackgroundWorker worker;
        private DateTime operationDateTime;
        private bool _debugMode = false;
        public bool DebugMode { get { return _debugMode; } set { _debugMode = value; } }

        public FTPClient(BackgroundWorker worker, DateTime time)
        {
            this.worker = worker;
            this.operationDateTime = time;
        }

        public List<TransmissionInfo> GetFileListing(Uri remoteBasePath, NetworkCredential creds)
        {
            if (remoteBasePath.Scheme != Uri.UriSchemeFtp)
                return null;

            FtpWebRequest request = (FtpWebRequest)FtpWebRequest.Create(remoteBasePath);
            request.Method = WebRequestMethods.Ftp.ListDirectory;
            request.KeepAlive = false;
            request.Credentials = creds;

            FtpWebResponse response = (FtpWebResponse)request.GetResponse();

            string listing = "";
            try
            {
                Stream responseStream = response.GetResponseStream();
                StreamReader reader = new StreamReader(responseStream);
                byte[] buffer = new byte[4096];
                while (responseStream.Read(buffer, 0, buffer.Length) > 0)
                {
                    string message = ASCIIEncoding.ASCII.GetString(buffer);
                    listing += message;
                }
                response.Close();
            }
            catch (Exception ex)
            {
                AppLog.WriteExceptionToLog(ex, null, true);
                return null;
            }

            string[] splitListing = listing.Split(new char[] { '\r', '\n' });
            List<TransmissionInfo> filesToDownload = new List<TransmissionInfo>();
            foreach (string s in splitListing)
            {
                string f = s;
                foreach (string seg in remoteBasePath.Segments)
                {
                    f = f.Replace(seg, "");
                }
                if (!f.Equals("") && !f.Trim(new char[] { '.', '\0' }).Equals(""))
                {
                    TransmissionInfo info = new TransmissionInfo();
                    info.FileName = f;

                    #region depr
                    //request = (FtpWebRequest)FtpWebRequest.Create(remoteBasePath.AbsoluteUri + "/" + f);
                    //request.Credentials = creds;
                    //request.Timeout = 120000;
                    //request.KeepAlive = false;
                    //request.UseBinary = true;
                    //request.Method = WebRequestMethods.Ftp.GetFileSize;
                    //long fileSize = 0;
                    //try
                    //{
                    //    response = (FtpWebResponse)request.GetResponse();
                    //    fileSize = response.ContentLength;
                    //}
                    //catch (Exception ex) 
                    //{
                    //    //IPHostEntry dns = Dns.GetHostEntry(remoteBasePath.Host);
                    //    //IPEndPoint serverIP = new IPEndPoint(dns.AddressList[0], 21);
                    //    //Socket sock = new Socket(AddressFamily.InterNetwork, SocketType.Stream, ProtocolType.Tcp);
                    //    //sock.Connect(serverIP);

                    //    //string[] commands = new string[]
                    //    //{
                    //    //    "USER " + creds.UserName,
                    //    //    "PASS " + creds.Password,
                    //    //    "SIZE " + 
                    //    //}
                    //    //string resp = "";
                    //    //byte[] receiveBuffer = new byte[2048];
                    //    //int bytes = 0;
                    //    //byte[] sendBuffer = Encoding.ASCII.GetBytes("USER " + creds.UserName);
                    //}

                    //info.FileSize = fileSize;
                    #endregion

                    filesToDownload.Add(info);
                }
            }

            return filesToDownload;
        }

        public void DownloadFiles(string localRootPath, Uri remotePath,
            ref List<TransmissionInfo> filesToDownload, NetworkCredential creds)
        {
            if (!localRootPath.EndsWith("\\"))
                localRootPath += "\\";
            long totalBytes = 0;
            long currentBytes = 0;
            string creationTimeString = this.operationDateTime.ToString("yyyyMMddHHmmss");

            foreach (TransmissionInfo info in filesToDownload)
                totalBytes += info.FileSize;

            FtpWebRequest request;
            FtpWebResponse response;

            int fileCount = 0;
            TransferProgress p = new TransferProgress();
            p.TotalBytes = totalBytes;
            p.State = OperationState.Download;
            p.TotalFiles = filesToDownload.Count;

            foreach (TransmissionInfo info in filesToDownload)
            {
                try
                {
                    request = (FtpWebRequest)FtpWebRequest.Create(remotePath.AbsoluteUri + "/" + info.FileName);
                    request.Method = WebRequestMethods.Ftp.DownloadFile;
                    request.KeepAlive = false;
                    request.Credentials = creds;
                    request.UseBinary = false; //okay as long as files are ASCII only

                    //TODO use original datetime
                    //request.Method = WebRequestMethods.Ftp.GetDateTimestamp;

                    response = (FtpWebResponse)request.GetResponse();
                    string localFilePath = localRootPath + string.Format("{0}_{1}", creationTimeString, info.FileName);
                    FileInfo local = new FileInfo(localFilePath);
                    using (Stream responseStream = response.GetResponseStream())
                    {
                        using (StreamReader reader = new StreamReader(responseStream))
                        {
                            if (!local.Directory.Exists)
                                local.Directory.Create();
                            using (FileStream fs = local.Create())
                            {
                                byte[] buffer = new byte[2048];
                                int bytesRead = 0;
                                p.FilesTransferred = ++fileCount;
                                p.CurrentFileName = info.FileName;

                                bytesRead = responseStream.Read(buffer, 0, buffer.Length);
                                while (bytesRead > 0)
                                {
                                    currentBytes += bytesRead;
                                    fs.Write(buffer, 0, bytesRead);
                                    bytesRead = responseStream.Read(buffer, 0, buffer.Length);
                                    p.BytesTransferred = currentBytes;
                                    int progress = Utils.CalculatePercentage(currentBytes, totalBytes);
                                    if (progress > 100)
                                        progress = 100;
                                    else if (progress < 0)
                                        progress = 0;
                                    worker.ReportProgress(progress, p);
                                }
                            }
                        }
                    }

                    
                    if (local.Exists && local.Length > 0)
                    {
                        info.LocalFile = local;
                        if (!_debugMode)
                            DeleteFile(remotePath, info.FileName, creds);
                    }
                }
                catch (Exception ex)
                {
                    AppLog.WriteExceptionToLog(ex, null, true);
                }
            }
        }

        public void DeleteFile(Uri remotePath, string fileName, NetworkCredential creds)
        {
            if (remotePath.Scheme != Uri.UriSchemeFtp)
                return;

            try
            {
                FtpWebRequest request = (FtpWebRequest)FtpWebRequest.Create(remotePath.AbsoluteUri + "/" + fileName);
                request.Method = WebRequestMethods.Ftp.DeleteFile;
                request.KeepAlive = false;
                request.Credentials = creds;

                FtpWebResponse response = (FtpWebResponse)request.GetResponse();
            }
            catch (Exception ex)
            {
                AppLog.WriteExceptionToLog(ex, null, true);
            }
        }

        public bool UploadFiles(Uri remotePath, ref List<TransmissionInfo> filesToUpload, NetworkCredential creds)
        {
            long totalBytes = 0;
            long currentBytes = 0;
            string creationTimeString = this.operationDateTime.ToString("yyyyMMddHHmmss");

            foreach (TransmissionInfo info in filesToUpload)
                totalBytes += info.LocalFile.Length;

            FtpWebRequest request;
            //FtpWebResponse response;

            int fileCount = 0;
            TransferProgress p = new TransferProgress();
            p.TotalBytes = totalBytes;
            p.State = OperationState.Upload;
            p.TotalFiles = filesToUpload.Count;
            bool success = true;

            foreach (TransmissionInfo info in filesToUpload)
            {
                try
                {
                    request = (FtpWebRequest)FtpWebRequest.Create(new Uri(remotePath.AbsoluteUri + "/" + info.FileName));
                    request.Method = WebRequestMethods.Ftp.UploadFile;
                    request.KeepAlive = false;
                    request.Credentials = creds;
                    request.ContentLength = info.LocalFile.Length;
                    request.UseBinary = false;

                    //response = (FtpWebResponse)request.GetResponse();
                    using (Stream requestStream = request.GetRequestStream())
                    {
                        using (FileStream fs = info.LocalFile.OpenRead())
                        {
                            byte[] buffer = new byte[2048];
                            int bytesRead = 0;
                            p.FilesTransferred = ++fileCount;
                            p.CurrentFileName = info.FileName;

                            bytesRead = fs.Read(buffer, 0, buffer.Length);
                            while (bytesRead != 0)
                            {
                                currentBytes += bytesRead;
                                requestStream.Write(buffer, 0, bytesRead);
                                bytesRead = fs.Read(buffer, 0, buffer.Length);
                                p.BytesTransferred = currentBytes;
                                int progress = (int)(((float)currentBytes / totalBytes) * 100f);
                                if (progress > 100)
                                    progress = 100;
                                else if (progress < 0)
                                    progress = 0;

                                worker.ReportProgress(progress, p);
                            }

                            fs.Close();
                        }
                        requestStream.Close();
                    }
                }
                catch (Exception ex)
                {
                    AppLog.WriteExceptionToLog(ex, null, true);
                    info.HasErrors = true;
                    if (info.UploadErrors == null)
                        info.UploadErrors = new List<UploadError>();
                    info.UploadErrors.Add(new UploadError() { Exception = ex });
                    success = false;
                }
            }

            return success;
        }
    }

    public class TransmissionInfo : IComparable<TransmissionInfo>
    {
        private List<UploadError> _upErrs = null;

        private long _fileSize = 0;
        private FileInfo _localFile;
        public string FileName { get; set; }
        public long FileSize
        {
            get
            {
                return _fileSize;
            }
            set
            {
                _fileSize = value;
            }
        }
        public FileInfo LocalFile
        {
            get
            {
                return _localFile;
            }
            set
            {
                _localFile = value;
                if (_localFile.Exists)
                    _fileSize = _localFile.Length;
            }
        }
        public bool HasErrors { get; set; }
        public List<UploadError> UploadErrors
        {
            get
            {
                return _upErrs;
            }
            set
            {
                _upErrs = value;
                HasErrors = true;
            }
        }

        public TransmissionInfo() { }
        public TransmissionInfo(string fileName)
        {
            FileName = fileName;
        }

        #region IComparable<TransmissionInfo> Members

        public int CompareTo(TransmissionInfo other)
        {
            int rtn = 0;
            FileTypeValue type1 = (FileTypeValue)Enum.Parse(typeof(FileTypeValue), this.FileName.Substring(0, 3));
            FileTypeValue type2 = (FileTypeValue)Enum.Parse(typeof(FileTypeValue), other.FileName.Substring(0, 3));

            int calc = type1 - type2;
            if (calc < 0)
                rtn = -1;
            else if (calc == 0)
            {
                rtn = this.FileName.CompareTo(other.FileName);
            }
            else
                rtn = 1;

            return rtn;
        }

        #endregion

        internal enum FileTypeValue
        {
            G80 = 0,
            G08 = 1,
            G73 = 2,
            G78 = 3,
            G70 = 4,
            G95 = 5,
            G07 = 6,
            G92 = 7,
            G96 = 8,
            G50 = 9,
            G51 = 10,
        }
    }

    public class TransmissionInfoComparer : IComparer<TransmissionInfo>
    {
        #region IComparer<TransmissionInfo> Members

        public int Compare(TransmissionInfo x, TransmissionInfo y)
        {
            return x.CompareTo(y);
        }

        #endregion
    }


    public class UploadError
    {
        private Exception _ex;
        public Exception Exception
        {
            get { return _ex; }
            set
            {
                _ex = value;
                Message += _ex.Message;
            }
        }
        public string Message { get; set; }
    }
}