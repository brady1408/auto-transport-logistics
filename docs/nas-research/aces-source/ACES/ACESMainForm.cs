using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Data;
using System.Drawing;
using System.Linq;
using System.Text;
using System.Windows.Forms;
using System.Threading;
using System.IO;
using System.Net;
using System.Globalization;

namespace ATLNetwork.EDI.ACES
{
    public partial class ACESMainForm : Form
    {
        private string connectionString;
        private const string ACES_DATA_FORMAT_TYPE = "ACES";
        private const string ICL_DATA_FORMAT_TYPE = "ICL";
        private DateTime operationDateTime = DateTime.Now;
        private bool conversionData = false;
        private DateTime convStart = DateTime.MinValue, convEnd = DateTime.MaxValue;
        private bool conversionDatesSpecified = false;
        private bool exitOnComplete = false;
        private bool debugMode = false;
        private bool deleteG80AfterProcess = true;

        public ACESMainForm(string cstr)
        {
            InitializeComponent();
            this.connectionString = cstr;
            progressBar_main.Minimum = 0;
            progressBar_main.Maximum = 100;
            progressBar_main.Step = 1;
        }

        public ACESMainForm(string cstr, bool conversion, bool exitOnComplete, bool debugMode)
        {
            InitializeComponent();
            this.connectionString = cstr;
            progressBar_main.Minimum = 0;
            progressBar_main.Maximum = 100;
            progressBar_main.Step = 1;
            this.conversionData = conversion;
            this.exitOnComplete = exitOnComplete;
            this.debugMode = debugMode;
        }

        public ACESMainForm(string cstr, bool conversion, DateTime convStart, DateTime convEnd, 
            bool exitOnComplete, bool debugMode)
        {
            InitializeComponent();
            this.connectionString = cstr;
            progressBar_main.Minimum = 0;
            progressBar_main.Maximum = 100;
            progressBar_main.Step = 1;
            this.conversionDatesSpecified = true;
            this.conversionData = conversion;
            this.convStart = convStart;
            this.convEnd = convEnd;
            this.exitOnComplete = exitOnComplete;
            this.debugMode = debugMode;
        }

        private void ACESMainForm_Load(object sender, EventArgs e)
        {
            BackgroundWorker sendReceiveWorker = new BackgroundWorker();
            sendReceiveWorker.WorkerSupportsCancellation = false;
            sendReceiveWorker.WorkerReportsProgress = true;
            sendReceiveWorker.DoWork += new DoWorkEventHandler(sendReceiveWorker_DoWork);
            sendReceiveWorker.ProgressChanged += new ProgressChangedEventHandler(sendReceiveWorker_ProgressChanged);
            sendReceiveWorker.RunWorkerCompleted += new RunWorkerCompletedEventHandler(sendReceiveWorker_RunWorkerCompleted);

            sendReceiveWorker.RunWorkerAsync();
        }

        void sendReceiveWorker_RunWorkerCompleted(object sender, RunWorkerCompletedEventArgs e)
        {
            label_status.Text = "Completed";
            try
            {
                //Utils.EmailErrorReport();
                Utils.LogAllErrors(new ATLDbDataContext(connectionString), connectionString);
            }
            catch (Exception ex)
            {
                AppLog.WriteExceptionToLog(ex, "Unable to email Log, see next entry", true);
                AppLog.WriteToLog(Utils.GetErrorList());
            }
            if (exitOnComplete)
            {
                this.Invalidate();
                this.Update();
                Thread.Sleep(500);
                Environment.Exit(0);
            }
            button_close.Enabled = true;
        }

        void sendReceiveWorker_ProgressChanged(object sender, ProgressChangedEventArgs e)
        {
            TransferProgress progress = e.UserState as TransferProgress;
            ProcessProgress pProg = e.UserState as ProcessProgress;
            if (progress != null)
            {
                string suffix1 = "B", suffix2 = "B";
                int bytes = (int)progress.BytesTransferred, tbytes = (int)progress.TotalBytes;
                if (progress.BytesTransferred > 1024)
                {
                    if (progress.BytesTransferred > 1024 * 1024)
                    {
                        suffix1 = "MB";
                        bytes = (int)progress.BytesTransferred / (1024 * 1024);
                    }
                    else
                    {
                        suffix1 = "KB";
                        bytes = (int)progress.BytesTransferred / 1024;
                    }
                }
                if (progress.TotalBytes > 1024)
                {
                    if (progress.TotalBytes > 1024 * 1024)
                    {
                        suffix2 = "MB";
                        tbytes = (int)progress.TotalBytes / (1024 * 1024);
                    }
                    else
                    {
                        suffix2 = "KB";
                        tbytes = (int)progress.TotalBytes / 1024;
                    }
                }

                string dir = progress.State == OperationState.Upload ? "Uploading" : "Downloading";
                label_status.Text = string.Format("{0} {1} ({2} of {3})...", dir, progress.CurrentFileName, progress.FilesTransferred, progress.TotalFiles);
                if (progress.TotalBytes > 0)
                {
                    label_progress.Text = string.Format("{0}{1} / {2}{3}", bytes, suffix1, tbytes, suffix2);
                    int progValue = (int)(((float)progress.BytesTransferred / progress.TotalBytes) * 100f);
                    if (progValue > 100)
                        progValue = 100;
                    else if (progValue < 0)
                        progValue = 0;
                    this.progressBar_main.Value = progValue;
                }
                else
                {
                    int progValue = (int)(((float)progress.FilesTransferred / progress.TotalFiles) * 100f);
                    if (progValue > 100)
                        progValue = 100;
                    else if (progValue < 0)
                        progValue = 0;

                    label_progress.Text = string.Format("{0}{1} {2}%", bytes, suffix1, progValue);

                    this.progressBar_main.Value = progValue;
                }
            }
            else if (pProg != null)
            {
                if (pProg.StatusMessage != null && pProg.StatusMessage != "" && pProg.State == OperationState.Query)
                {
                    label_status.Text = pProg.StatusMessage;
                    label_progress.Text = string.Format("{0}%", e.ProgressPercentage);
                    progressBar_main.Value = e.ProgressPercentage;
                }
                else if (pProg.State == OperationState.Query)
                {
                    label_status.Text = "Querying...";
                    label_progress.Text = "";
                    progressBar_main.Value = 0;
                }
                else if (pProg.StatusMessage != null && pProg.StatusMessage != "" && pProg.State == OperationState.Processing)
                {
                    label_status.Text = pProg.StatusMessage;
                    progressBar_main.Value = e.ProgressPercentage;
                    label_progress.Text = string.Format("{0}%", e.ProgressPercentage);
                }
                else if (pProg.State == OperationState.Processing)
                {
                    label_status.Text = string.Format("Processing {0}...({1} of {2})",
                        pProg.CurrentFileName,
                        pProg.FilesProcessed,
                        pProg.TotalFiles);
                    label_progress.Text = string.Format("{0}%", e.ProgressPercentage);
                    progressBar_main.Value = e.ProgressPercentage;
                }
                else
                {
                    label_status.Text = pProg.StatusMessage;
                    label_progress.Text = "";
                    progressBar_main.Value = e.ProgressPercentage;
                }
            }
            else
            {
                label_status.Text = "Transmitting...";
                label_progress.Text = string.Format("{0}%", e.ProgressPercentage);
                this.progressBar_main.Value = e.ProgressPercentage;
            }
        }

        void sendReceiveWorker_DoWork(object sender, DoWorkEventArgs e)
        {
            BackgroundWorker worker = sender as BackgroundWorker;
            TransferProgress tp = new TransferProgress();
            ProcessProgress pp = new ProcessProgress();
            string uploadArchivePath = string.Format("{0}\\Upload_Backup\\", ACES_DATA_FORMAT_TYPE);
            string downloadArchivePath = string.Format("{0}\\Download_Backup\\", ACES_DATA_FORMAT_TYPE);

            ATLDbDataContext db = new ATLDbDataContext(connectionString);

            if (debugMode)
            {
                TextWriter log;
                log = File.AppendText("acesquerylog.sql");
                db.Log = log;

                Utils.CheckErrorTableExists(db, connectionString);
            }

            X00 x00 = (from p in db.X00s
                       where p.DataFormatType.Equals(ACES_DATA_FORMAT_TYPE)
                       select p).FirstOrDefault() as X00;

            string upUsr, upPwd, dnUsr, dnPwd;
            upUsr = x00.UpUserId;
            upPwd = x00.UpPassword;
            dnUsr = x00.DnUserId;
            dnPwd = x00.DnPassword;
            string upSvr = (x00.UpServerName.StartsWith("ftp://", true, CultureInfo.InvariantCulture) ? x00.UpServerName : "ftp://" + x00.UpServerName);
            string dnSvr = (x00.DnServerName.StartsWith("ftp://", true, CultureInfo.InvariantCulture) ? x00.DnServerName : "ftp://" + x00.DnServerName);
            if (upSvr.Equals("ftp://") && !dnSvr.Equals("ftp://"))
                upSvr = dnSvr;
            else if (dnSvr.Equals("ftp://") && !upSvr.Equals("ftp://"))
                dnSvr = upSvr;
            Uri upPath = new Uri(new Uri(upSvr), x00.UpPath);
            NetworkCredential upCreds = new NetworkCredential(upUsr, upPwd);
            Uri dnPath = new Uri(new Uri(dnSvr), x00.DnPath);
            NetworkCredential dnCreds = new NetworkCredential(dnUsr, dnPwd);

            FTPClient ftp = new FTPClient(worker, operationDateTime);
            ftp.DebugMode = debugMode;

            if (x00.UploadData == null || (byte)1 == (byte)x00.UploadData)
            {
                pp.StatusMessage = "Generating R41s...";
                pp.State = OperationState.Query;
                worker.ReportProgress(0, pp);
                List<R41> r41s = R41.Generate(db, worker, operationDateTime);
                if (r41s != null)
                {
                    foreach (R41 r41 in r41s)
                        r41.CreatedDateTime = operationDateTime;
                }

                pp.StatusMessage = "Generating R92s...";
                pp.State = OperationState.Query;
                worker.ReportProgress(0, pp);
                List<R92> r92s = R92.Generate(db, worker, operationDateTime);
                if (r92s != null)
                {
                    foreach (R92 r92 in r92s)
                        r92.CreatedDateTime = operationDateTime;
                }

                pp.StatusMessage = "Preparing to upload...";
                worker.ReportProgress(100, pp);
                List<TransmissionInfo> uploadFiles = null;

                try
                {
                    uploadFiles = PrepareUploads(uploadArchivePath, db, r41s, r92s);
                }
                catch (Exception ex)
                {
                    AppLog.WriteExceptionToLog(ex, "Unable to create upload files:", false);
                }

                if (uploadFiles != null && uploadFiles.Count > 0)
                {
                    if (!ftp.UploadFiles(upPath, ref uploadFiles, upCreds))
                    {
                        string upErrMsg = "Upload errors occurred: " + Environment.NewLine;
                        foreach (TransmissionInfo info in uploadFiles)
                        {
                            if (info.HasErrors && info.UploadErrors != null)
                            {
                                foreach (UploadError err in info.UploadErrors)
                                {
                                    upErrMsg += string.Format(">>\t{1}{2}", err.Message, Environment.NewLine);
                                }
                            }
                        }
                        AppLog.WriteToLog(upErrMsg);
                    }
                    if (debugMode)
                        MarkUploadProcessed(db, r41s, r92s, 0, worker);
                    else
                        MarkUploadProcessed(db, r41s, r92s, 1, worker);
                }
                else
                {
                    if ((r41s != null && r41s.Count > 0) || (r92s != null && r92s.Count > 0))
                    {
                        string msg = "Failed to create upload files for the following messages:" + Environment.NewLine;
                        if (r41s != null)
                            foreach (R41 r in r41s)
                                foreach (R41Detail d in r.Detail)
                                    msg += string.Format("(R41){0}{1}", d.VIN, Environment.NewLine);
                        if (r92s != null)
                            foreach (R92 r in r92s)
                                foreach (R92Detail d in r.Detail)
                                    msg += string.Format("(R92){0}{1}", d.VIN, Environment.NewLine);
                        AppLog.WriteToLog(msg);
                        MarkUploadProcessed(db, r41s, r92s, 0, worker);
                    }
                }
            }

            if (x00.DownloadData == null || (byte)1 == (byte)x00.DownloadData)
            {
                pp.StatusMessage = "Preparing to download...";
                worker.ReportProgress(0, pp);

                List<TransmissionInfo> downloadFiles = ftp.GetFileListing(dnPath, dnCreds);
                if (downloadFiles.Count > 0)
                {
                    ftp.DownloadFiles(downloadArchivePath, dnPath, ref downloadFiles, dnCreds);
                    downloadFiles.Sort(new TransmissionInfoComparer());
                    ProcessReceivedFiles(db, x00.X00Id, downloadFiles, worker);
                }
            }

            if (conversionData)
            {
                //send conversion data
                pp.StatusMessage = "Generating conversion data...";
                pp.State = OperationState.Query;
                worker.ReportProgress(0, pp);
                SHS shs = SHS.Generate(db, worker, operationDateTime);

                if (shs != null)
                {
                    pp.StatusMessage = "Preparing to upload...";
                    worker.ReportProgress(100, pp);
                    int seqNum = 0;
                    X00 convX00 = null;
                    try
                    {
                        convX00 = (from p in db.X00s
                                   where p.SystemName.Equals("ACES Conversion") &&
                                        p.DataFormatType.Equals("_ACES_C")
                                   select p).FirstOrDefault() as X00;
                    }
                    catch (Exception ex) { AppLog.WriteExceptionToLog(ex, null, true); }

                    if (convX00 != null && ((byte)convX00.UpActive == (byte)1))
                    {
                        seqNum = Utils.GetNextSequenceNumber(db);
                        string convUsr, convPwd;
                        convUsr = convX00.UpUserId;
                        convPwd = convX00.UpPassword;
                        string convSvr = (convX00.UpServerName.StartsWith("ftp://", true, CultureInfo.InvariantCulture) ?
                            convX00.UpServerName : "ftp://" + convX00.UpServerName);
                        Uri convPath = new Uri(new Uri(convSvr), convX00.UpPath);
                        NetworkCredential convCreds = new NetworkCredential(convUsr, convPwd);

                        TransmissionInfo tiSHS = new TransmissionInfo(shs.GetFileName(seqNum));
                        string shsPath = string.Format("{0}{1}_{2}",
                            uploadArchivePath + "Conversion\\",
                            operationDateTime.ToString("yyyyMMddHHmmss"),
                            tiSHS.FileName);

                        shs.Save(shsPath);
                        tiSHS.LocalFile = new FileInfo(shsPath);
                        shs.TransmissionInfo = tiSHS;

                        List<TransmissionInfo> conversionList = new List<TransmissionInfo>(1);
                        conversionList.Add(tiSHS);
                        ftp.UploadFiles(convPath, ref conversionList, convCreds);
                    }
                }
            }
        }

        private void ProcessReceivedFiles(ATLDbDataContext db, int x00Id, List<TransmissionInfo> downloadFiles,
            BackgroundWorker worker)
        {
            ProcessProgress p = new ProcessProgress();
            int filesProcessed = 0;
            int totalFiles = downloadFiles.Count;

            p.TotalFiles = totalFiles;
            p.FilesProcessed = filesProcessed;
            p.State = OperationState.Processing;

            foreach (TransmissionInfo info in downloadFiles)
            {
                p.CurrentFileName = info.FileName;
                p.FilesProcessed = ++filesProcessed;
                p.StatusMessage = "";

                worker.ReportProgress(Utils.CalculatePercentage(filesProcessed, totalFiles), p);
                try
                {
                    switch (info.FileName.Substring(0, 3))
                    {
                        case "G07":
                            G07 g07 = G07.Load(info);
                            g07.CreatedDateTime = operationDateTime;
                            g07.Process(db);
                            break;
                        case "G08":
                            G08 g08 = G08.Load(info);
                            g08.CreatedDateTime = operationDateTime;
                            g08.Process(db);
                            break;
                        case "G50":
                            G50 g50 = G50.Load(info);
                            g50.CreatedDateTime = operationDateTime;
                            g50.Process(db);
                            foreach (G50Detail d in g50.Detail)
                            {
                                Error err = new Error();
                                err.Message = "ACES Rejection";
                                err.Description = d.RejectExplanation.Value;
                                err.Code = d.RejectCode.Value;
                                err.EdiSet = d.FileNameToResend.Value.Substring(0, 3);
                                err.System = "ACES";
                                err.ErrorDateTime = operationDateTime;
                                err.FilePath = d.FileNameToResend.Value + "." + d.FileNameExtension.Value;
                                err.Detail = d.ToString();
                                err.Active = true;

                                Utils.AddErrorEntry(err);
                            }
                            break;
                        case "G51":
                            G51 g51 = G51.Load(info);
                            g51.CreatedDateTime = operationDateTime;
                            g51.Process(db);
                            foreach (G51Detail d in g51.Detail)
                            {
                                Error err = new Error();
                                err.Message = "ACES Rejection";
                                err.Description = d.ErrorCodeDescription.Value;
                                err.Code = d.ErrorCode.Value;
                                err.EdiSet = d.FileName.Value.Substring(0, 3);
                                err.System = "ACES";
                                err.ErrorDateTime = operationDateTime;
                                err.FilePath = d.FileName.Value;
                                err.VIN = d.VIN.Value;
                                err.Detail = d.ToString();
                                err.Active = true;

                                Utils.AddErrorEntry(err);
                            }
                            break;
                        case "G70":
                            G70 g70 = G70.Load(info);
                            g70.CreatedDateTime = operationDateTime;
                            g70.Process(db);
                            break;
                        case "G73":
                            G73 g73 = G73.Load(info);
                            g73.CreatedDateTime = operationDateTime;
                            g73.Process(db);
                            break;
                        case "G78":
                            G78 g78 = G78.Load(info);
                            g78.CreatedDateTime = operationDateTime;
                            g78.Process(db);
                            break;
                        case "G80":
                            G80 g80 = G80.Load(info);
                            g80.CreatedDateTime = operationDateTime;
                            if (g80.Process(db) && deleteG80AfterProcess)
                            {
                                try
                                {
                                    g80.TransmissionInfo.LocalFile.Delete();
                                }
                                catch (Exception ex)
                                {
                                    AppLog.WriteExceptionToLog(ex, null, true);
                                }
                            }
                            else
                            {
                                try
                                {
                                    g80.TransmissionInfo.LocalFile.CopyTo(string.Format("ACES\\Pending\\{0}",
                                        g80.TransmissionInfo.LocalFile.Name));
                                }
                                catch (Exception ex)
                                {
                                    AppLog.WriteExceptionToLog(ex, null, true);
                                }
                            }
                            break;
                        case "G92":
                            G92 g92 = G92.Load(info);
                            g92.CreatedDateTime = operationDateTime;
                            if (!g92.Process(db))
                            {
                                try
                                {
                                    g92.TransmissionInfo.LocalFile.CopyTo(string.Format("ACES\\Pending\\{0}",
                                        g92.TransmissionInfo.LocalFile.Name));
                                }
                                catch (Exception ex)
                                {
                                    AppLog.WriteExceptionToLog(ex, null, true);
                                }
                            }
                            break;
                        case "G95":
                            G95 g95 = G95.Load(info);
                            g95.CreatedDateTime = operationDateTime;
                            if (!g95.Process(db))
                            {
                                try
                                {
                                    g95.TransmissionInfo.LocalFile.CopyTo(string.Format("ACES\\Pending\\{0}",
                                        g95.TransmissionInfo.LocalFile.Name));
                                }
                                catch (Exception ex)
                                {
                                    AppLog.WriteExceptionToLog(ex, null, true);
                                }
                            }
                            break;
                        case "G96":
                            G96 g96 = G96.Load(info);
                            g96.CreatedDateTime = operationDateTime;
                            if (!g96.Process(db))
                            {
                                try
                                {
                                    g96.TransmissionInfo.LocalFile.CopyTo(string.Format("ACES\\Pending\\{0}",
                                        g96.TransmissionInfo.LocalFile.Name));
                                }
                                catch (Exception ex)
                                {
                                    AppLog.WriteExceptionToLog(ex, null, true);
                                }
                            }
                            break;
                        default:
                            break;
                    }
                }
                catch (Exception ex)
                {
                    AppLog.WriteExceptionToLog(ex, null, true);
                }

                worker.ReportProgress(Utils.CalculatePercentage(filesProcessed, totalFiles), p);
            }
        }

        private void MarkUploadProcessed(ATLDbDataContext db, List<R41> r41s, List<R92> r92s, int status, BackgroundWorker worker)
        {
            ProcessProgress pp = new ProcessProgress();
            if (r41s != null)
            {
                foreach (R41 r41 in r41s)
                {
                    if (r41 != null && !r41.TransmissionInfo.HasErrors)
                    {
                        int total = r41.Detail.Count;
                        int count = 0;
                        foreach (R41Detail d in r41.Detail)
                        {
                            count++;
                            X85 x85 = d.X85;
                            if (x85 == null)
                            {
                                x85 = (from p in db.X85s
                                       where p.X85Id == d.X85Id
                                       select p).FirstOrDefault() as X85;
                            }

                            if (x85 == null)
                            {
                                AppLog.WriteToLog("Unable to locate specified X85 = " + d.X85.X85Id);
                            }

                            pp.StatusMessage = string.Format("Processing R41 {0} of {1}", count, total);
                            pp.State = OperationState.Processing;
                            int prog = (int)(((float)count / total) * 100f);
                            if (prog > 100)
                                prog = 100;
                            else if (prog < 0)
                                prog = 0;
                            worker.ReportProgress(prog, pp);
                            if (x85 != null)
                            {
                                x85.Processed = (byte)status;
                                x85.ProcessedTimeString = operationDateTime;
                                x85.UpFileName = r41.TransmissionInfo.LocalFile.Name;
                            }
                            try
                            {
                                db.SubmitChanges(System.Data.Linq.ConflictMode.ContinueOnConflict);
                                count++;
                            }
                            catch (Exception ex)
                            {
                                AppLog.WriteExceptionToLog(ex, "Concurrency Exception (R41)", false);
                            }
                        }
                    }
                    else if (r41.TransmissionInfo.HasErrors)
                    {
                        AppLog.WriteToLog("Errors detected in R92 upload.");
                    }
                }
            }

            if (r92s != null)
            {
                foreach (R92 r92 in r92s)
                {
                    if (r92 != null && !r92.TransmissionInfo.HasErrors)
                    {
                        int total = r92.Detail.Count;
                        int count = 0;
                        foreach (R92Detail d in r92.Detail)
                        {
                            X85 x85 = d.X85;
                            if (x85 == null)
                            {
                                x85 = (from p in db.X85s
                                       where p.X85Id == d.X85Id
                                       select p).FirstOrDefault() as X85;
                            }

                            pp.StatusMessage = string.Format("Processing R92 {0} of {1}", count, total);
                            pp.State = OperationState.Processing;
                            int prog = Utils.CalculatePercentage(count, total);
                            if (prog > 100)
                                prog = 100;
                            else if (prog < 0)
                                prog = 0;
                            worker.ReportProgress(prog, pp);
                            if (x85 != null)
                            {
                                x85.Processed = (byte)status;
                                x85.ProcessedTimeString = operationDateTime;
                                x85.UpFileName = r92.TransmissionInfo.LocalFile.Name;
                            }
                            try
                            {
                                db.SubmitChanges(System.Data.Linq.ConflictMode.ContinueOnConflict);
                                count++;
                            }
                            catch (Exception ex)
                            {
                                AppLog.WriteExceptionToLog(ex, "Concurrency Exception (R92)", false);
                            }
                        }
                    }
                    else if (r92.TransmissionInfo.HasErrors)
                    {
                        AppLog.WriteToLog("Errors detected in R92 upload.");
                    }
                }
            }
        }

        private List<TransmissionInfo> PrepareUploads(string uploadArchivePath, ATLDbDataContext db, List<R41> r41s, List<R92> r92s)
        {
            List<TransmissionInfo> uploadFiles = new List<TransmissionInfo>();
            if (r41s != null)
            {
                foreach (R41 r41 in r41s)
                {
                    if (r41 != null && r41.Detail.Count > 0)
                    {
                        TransmissionInfo tiR41 = new TransmissionInfo(r41.GetFileName(Utils.GetNextSequenceNumber(db)));
                        string r41Path = string.Format("{0}{1}_{2}",
                            uploadArchivePath,
                            operationDateTime.ToString("yyyyMMddHHmmss"),
                            tiR41.FileName);

                        r41.Save(r41Path);
                        tiR41.LocalFile = new FileInfo(r41Path);
                        r41.TransmissionInfo = tiR41;
                        uploadFiles.Add(tiR41);
                    }
                }
            }

            if (r92s != null)
            {
                foreach (R92 r92 in r92s)
                {
                    if (r92 != null && r92.Detail.Count > 0)
                    {
                        TransmissionInfo tiR92 = new TransmissionInfo(r92.GetFileName(Utils.GetNextSequenceNumber(db)));
                        string r92Path = string.Format("{0}{1}_{2}",
                            uploadArchivePath,
                            operationDateTime.ToString("yyyyMMddHHmmss"),
                            tiR92.FileName);

                        r92.Save(r92Path);
                        tiR92.LocalFile = new FileInfo(r92Path);
                        r92.TransmissionInfo = tiR92;
                        uploadFiles.Add(tiR92);
                    }
                }
            }

            return uploadFiles;
        }

        private void button_close_Click(object sender, EventArgs e)
        {
            Environment.Exit(0);
        }
    }

    public class TransferProgress
    {
        public long TotalBytes { get; set; }
        public long BytesTransferred { get; set; }
        public int TotalFiles { get; set; }
        public int FilesTransferred { get; set; }
        public OperationState State { get; set; }
        public string CurrentFileName { get; set; }
    }

    public class ProcessProgress
    {
        public string StatusMessage { get; set; }
        public OperationState State { get; set; }
        public int FilesProcessed { get; set; }
        public int TotalFiles { get; set; }
        public string CurrentFileName { get; set; }
    }

    public enum OperationState
    {
        Upload,
        Download,
        Query,
        GetFileListing,
        Processing
    }
}
