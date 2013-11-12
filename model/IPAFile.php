<?php
require_once APP_ROOT.'/libs/CFPropertyList/classes/CFPropertyList/CFPropertyList.php';

class IPAFile {

	protected function unzipFile($ipafile,$filename)
	{
		$p = popen("unzip -cq \"$ipafile\" \"$filename\" 2>/dev/null",'r');
		$ret = stream_get_contents($p);
		pclose($p);

		return $ret;
	}

	public static function parseInfoPlist($ipafile)
	{
		$info_plist = self::unzipFile($ipafile,'Payload/*/Info.plist');

		$plutil = new CFPropertyList\CFPropertyList();
		$plutil->parse($info_plist);
		return $plutil->toArray();
	}

}