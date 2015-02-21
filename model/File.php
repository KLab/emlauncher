re_once APP_ROOT.'/model/Config.php';

class File {

        public static function uploadData($key, $data, $type, $acl='private')
        {
                $config = Config::get('file');
                $dir = dirname($config["path"] . $key);
                if( !is_dir($dir)) {
                        mkdir($dir,  0777,  true);
                }
                return file_put_contents ( $config["path"] . $key,  $data);
        }

        public static function uploadFile($key, $filename, $type, $acl='private')
        {
                $config = Config::get('file');
                $dir = dirname($config["path"] . $key);
                if( !is_dir($dir)) {
                        mkdir($dir,  0777,  true);
                }
                return rename($filename,  $config["path"] . "/".$key);
        }

        public static function rename($srckey, $dstkey, $acl='private')
        {
                $config = Config::get('file');
                $dir = dirname($config["path"] . $srckey);
                if( !is_dir($dir)) {
                        mkdir($dir,  0777,  true);
                }
                $dir = dirname($config["path"] . $dstkey);
                if( !is_dir($dir)) {
                        mkdir($dir,  0777,  true);
                }
                return rename($config["path"] . "/". $srckey,  $config["path"] . "/". $dstkey);
        }

        public static function delete($key)
        {

                $config = Config::get('file');
                return unlink($config["path"] . "/". $key);
        }

        public static function url($key, $expires=null)
        {
                $config = Config::get('file');
                return $config["url"] . "/". $key;
        }

}


