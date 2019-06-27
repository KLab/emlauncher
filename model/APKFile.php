<?php

class APKFile {

	public static function getPackageName($apkfile)
	{
		$apk = new ApkParser\Parser($apkfile);
		return $apk->getManifest()->getPackageName();
	}

}
