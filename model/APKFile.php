<?php
require_once APP_ROOT.'/model/Config.php';

class APKFile {

	protected static $config = null;

	public static function getConfig(){
		if(static::$config===null){
			static::$config = Config::get('apkfile');
		}
		return static::$config;
	}

	public static function getPackageName($apkname)
	{
		$apk = new ApkParser\Parser($apkname);
		return $apk->getManifest()->getPackageName();
	}

	public static function extractFromAppBundle($aabname)
	{
		$apkname = tempnam("/tmp","apk");
		$apksname = "{$apkname}.apks";
		try{
			self::buildApks($aabname, $apksname);
			self::extractUniversalApk($apksname, $apkname);
			unlink($apksname);
		}
		catch(Exception $e){
			unlink($apkname);
			unlink($apksname);
			throw $e;
		}

		return $apkname;
	}

	private static function buildApks($aabname, $apksname)
	{
		$conf = self::getConfig();

		$java = $conf['java'] ?? 'java';

		$cmd = "{$java} -jar \"{$conf['bundletool']}\" build-apks --mode=universal".
			" --bundle=\"{$aabname}\"".
			" --output=\"{$apksname}\"".
			" --ks=\"{$conf['keystore']}\"".
			" --ks-pass=\"{$conf['kspass']}\"".
			" --ks-key-alias=\"{$conf['keyalias']}\"".
			" --key-pass=\"{$conf['keypass']}\"";
		if(!empty($conf['aapt2'])){
			$cmd .= " --aapt2=\"{$conf['aapt2']}\"";
		}
		exec($cmd, $out, $ret);
		if($ret!=0){
			throw new RuntimeException("bundletool error: ".implode("\n", $out));
		}
	}

	private static function extractUniversalApk($apksname, $apkname)
	{
		$zip = new ZipArchive();
		$r = $zip->open($apksname);
		if($r!==TRUE){
			throw new RuntimeException("ZipArchive::open failed: ".$r);
		}

		$strm = $zip->getStream('universal.apk');
		if($strm===FALSE){
			throw new RuntimeException("ZipArchive::getStream failed");
		}

		$apk = fopen($apkname, "w");
		if($apk===FALSE){
			throw new RuntimeException("fopen failed");
		}
		if(stream_copy_to_stream($strm, $apk)===FALSE){
			throw new RuntimeException("stream_copy_to_stream failed");
		}
		if(fclose($apk)===FALSE){
			throw new RuntimeException("fclose failed");
		}
		if($zip->close()===FALSE){
			throw new RuntimeException("zip close failed");
		}
	}

	public static function getKeystoreInfo()
	{
		$conf = self::getConfig();

		list($passtype, $pass) = explode(':',$conf['kspass'],2);
		switch($passtype){
		case 'pass':
			break;
		case 'file':
			$pass = file_get_contents($pass);
			if($pass===FALSE){
				throw new RuntimeException("file_get_contents failed");
			}
			break;
		default:
			throw new RuntimeException("invalid config format: apkfile.keypass");
		}

		$cmd = "keytool -list -v -keystore {$conf['keystore']} -alias {$conf['keyalias']} -storepass {$pass}";
		exec($cmd, $info);

		return $info;
	}
}
