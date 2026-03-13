using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.IO;

namespace ATLNetwork.EDI.ACES
{
    public static class AppLog
    {
        public static string LogFilePath = "aces.log";

        public static void WriteToLog(string entry)
        {
            try
            {
                //app_path = (new FileInfo(Application.ExecutablePath)).DirectoryName + "\\";
                //path = app_path + "log.txt";
                int sub = 0;

                while (File.Exists(LogFilePath))
                {
                    FileInfo f = new FileInfo(LogFilePath);
                    if (f.Length < 209715200) //If log file is 10MB or greater, create new file
                        break;
                    LogFilePath = "aces" + (++sub).ToString() + ".log";
                }

                entry = string.Format("[{0}]{1}{2}{3}{4}",
                    DateTime.Now,
                    Environment.NewLine,
                    entry,
                    Environment.NewLine,
                    Environment.NewLine);
                File.AppendAllText(LogFilePath, entry);
                Console.WriteLine("!!---------!!");
                Console.WriteLine(entry);
                Console.WriteLine("!!---------!!");
            }
            catch
            {
                Console.WriteLine("Unable to write to " + LogFilePath);
            }
        }

        public static void WriteExceptionToLog(Exception ex)
        {
            WriteExceptionToLog(ex, null, false);
        }

        public static void WriteExceptionToLog(Exception ex, string prefix, bool strackTrace)
        {
            WriteToLog(string.Format("{0}{1}Exception{2}Target: {3}{4}Source: {5}{6}Message: {7}{8}Inner Exception: {9}{10}{11}",
                (prefix == null || prefix.Equals("") ? "" : prefix),
                (prefix == null || prefix.Equals("") ? "" : Environment.NewLine),
                Environment.NewLine,
                ex.TargetSite,
                Environment.NewLine,
                ex.Source,
                Environment.NewLine,
                ex.Message,
                Environment.NewLine,
                (ex.InnerException != null ? ex.InnerException.Message : ""),
                (strackTrace ? Environment.NewLine : ""),
                (strackTrace ? ex.StackTrace : "")));
        }
    }
}
