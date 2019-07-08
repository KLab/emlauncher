<?php
require_once APP_ROOT.'/libs/CFPropertyList/classes/CFPropertyList/CFPropertyList.php';

class IPAFile {

	protected function unzipInfoPlistFileName($ipafile)
	{
		$p = popen("unzip -l \"$ipafile\" \"Payload/*/Info.plist\" 2>/dev/null",'r');
		$fname = null;
		while(($l=fgets($p))){
			if(preg_match('/ +(Payload\/[^\/]+\.app\/Info.plist)\n$/',$l,$m)){
				$fname = $m[1];
				break;
			}
		}
		pclose($p);

		return $fname;
	}

	protected function unzipFile($ipafile,$filename)
	{
		$p = popen("unzip -cq \"$ipafile\" \"$filename\" 2>/dev/null",'r');
		$ret = stream_get_contents($p);
		pclose($p);

		return $ret;
	}

	public static function parseInfoPlist($ipafile)
	{
		$plist_name = self::unzipInfoPlistFileName($ipafile);
		if(!$plist_name){
			throw new UnexpectedValueException(__METHOD__.": Info.plist file not found.");
		}
		$info_plist = self::unzipFile($ipafile,$plist_name);

		$plutil = new CFPropertyList\CFPropertyList();
		$plutil->parse($info_plist);
		return $plutil->toArray();
	}

}