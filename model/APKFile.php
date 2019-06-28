<?php

class APKFile {

	public static function getPackageName($apkname)
	{
		$apk = new ApkParser\Parser($apkname);
		return $apk->getManifest()->getPackageName();
	}

	public static function extractFromAppBundle($aabname)
	{
		$apkname = tempnam("/tmp","");
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
		$bundletool = "/bundletool.jar";
		$keystore = "/emlauncher.keystore";
		$kspass = "pass:emlauncher";
		$ksalias = "emlauncher";
		$keypass = "pass:emlauncher";

		$cmd = "java -jar \"{$bundletool}\" build-apks --mode=universal".
			" --bundle=\"{$aabname}\"".
			" --output=\"{$apksname}\"".
			" --ks=\"{$keystore}\"".
			" --ks-pass=\"{$kspass}\"".
			" --ks-key-alias=\"{$ksalias}\"".
			" --key-pass=\"{$keypass}\"";
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
	}
}
