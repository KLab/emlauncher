<?php
require_once APP_ROOT.'/model/Config.php';

require_once APP_ROOT.'/model/S3.php';
require_once APP_ROOT.'/model/File.php';

class Storage {

        protected static function getClass()
        {
                $storage= Config::get('storage');
                $config = Config::get( $storage["type"] );
                return $config["class"];
        }

        public static function uploadData($key, $data, $type, $acl='private')
        {
                $class = Storage::getClass();
                return $class::uploadData($key, $data, $type, $acl);
        }

        public static function uploadFile($key, $filename, $type, $acl='private')
        {
                $class = Storage::getClass();
                return $class::uploadFile($key, $filename, $type, $acl);
        }

        public static function rename($srckey, $dstkey, $acl='private')
        {
                $class = Storage::getClass();
                return $class::rename($srckey, $dstkey, $acl);
        }

        public static function delete($key)
        {
                $class = Storage::getClass();
                return $class::delete($key);
        }

        public static function url($key, $expires=null)
        {
                $class = Storage::getClass();
                return $class::url($key, $expires);
        }
}
