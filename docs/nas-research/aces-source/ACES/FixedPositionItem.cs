using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;

namespace ATLNetwork.EDI.ACES
{
    public class FixedPositionItem<T>
    {
        private T _value;
        private int _offset;
        private int _length;
        private string _format;
        private StringJustification _justification = StringJustification.Left;
        private bool _required = false;
        private bool _isFiller = false;
        private int _dbLength = 0;

        public T Value
        {
            get { return _value; }
            set 
            { 
                _value = value;
                if (_required && !_isFiller && (value == null || value.ToString().Equals("")))
                {
                    //throw new FileValidationException("Missing required data");
                }
            }
        }

        public int DBLength
        {
            get { return _dbLength; }
            set { _dbLength = value; }
        }

        public bool IsFiller
        {
            get { return _isFiller; }
            set { _isFiller = value; }
        }

        public int Offset
        {
            get { return _offset; }
            set { _offset = value; }
        }

        public int Length
        {
            get { return _length; }
            set { _length = value; }
        }

        public string Format
        {
            get { return _format; }
            set { _format = value; }
        }

        public StringJustification Justification
        {
            get { return _justification; }
            set { _justification = value; }
        }

        public bool Required
        {
            get { return _required; }
            set { _required = value; }
        }

        public override string ToString()
        {
            string f = _format;
            if (f == null || f.Equals(string.Empty))
            {
                f = "{0," + (_justification == StringJustification.Left ? "-" : "") + _length.ToString() + "}";
            }
            string fullVal = "";
            if (Value.GetType() == typeof(bool))
            {
                int i = (bool)(Value as bool?) ? 1 : 0;
                fullVal = string.Format(f, i);
                if (fullVal.Length > _length)
                {
                    return fullVal.Substring(0, _length);
                }
                else
                    return fullVal;
            }
            fullVal = string.Format(f, _value);
            if (fullVal.Length > _length)
            {
                return fullVal.Substring(0, _length);
            }
            else
                return fullVal;
        }

        public string ToString(int trim)
        {
            return this.ToString().Substring(0, trim);
        }

        public string ToString(bool doDBTrim)
        {
            if (doDBTrim && (DBLength > 0))
                return this.ToString(DBLength);
            else
                return this.ToString();
        }
    }

    public enum StringJustification
    {
        Left,
        Right
    }
}
