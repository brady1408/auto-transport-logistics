using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;

namespace ATLNetwork.EDI.ACES
{
    public class FileValidationException : Exception
    {
        public FileValidationException() : base() { }
        public FileValidationException(string message) : base(message) { }
        public FileValidationException(string message, Exception innerException) : base(message, innerException) { }
    }

    public class RecordCountMismatch : Exception
    {
        public RecordCountMismatch() : base() { }
        public RecordCountMismatch(string message) : base(message) { }
        public RecordCountMismatch(string message, Exception innerException) : base(message, innerException) { }
    }
}
