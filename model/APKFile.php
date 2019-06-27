<?php

class APKFile {

	public static function getPackageName($apkfile)
	{
		$apk = new ApkParser\Parser($apkfile);
        sleep(20);
		return $apk->getManifest()->getPackageName();
	}

}
