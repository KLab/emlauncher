<?php
require_once APP_ROOT.'/model/Random.php';
require_once APP_ROOT.'/model/S3.php';

/**
 * Row object for 'package' table.
 */
class Package extends mfwObject {
	const DB_CLASS = 'PackageDb';
	const SET_CLASS = 'PackageSet';

	const PF_ANDROID = 'Android';
	const PF_IOS = 'iOS';
	const PF_UNKNOWN = 'unknown';
	const MIME_ANDROID = 'application/vnd.android.package-archive';
	const MIME_IOS = 'application/octet-stream';


}

/**
 * Set of Package objects.
 */
class PackageSet extends mfwObjectSet {
	public static function hypostatize(Array $row=array())
	{
		return new Package($row);
	}
	protected function unsetCache($id)
	{
		parent::unsetCache($id);
	}
}

/**
 * database accessor for 'package' table.
 */
class PackageDb extends mfwObjectDb {
	const TABLE_NAME = 'package';
	const SET_CLASS = 'PackageSet';

	const TEMP_DIR = 'temp-data/';

	public static function uploadTemporary($name,$file,$mime)
	{
		$platform = Package::PF_UNKNOWN;
		$ext = pathinfo($name,PATHINFO_EXTENSION);

		switch(strtolower($ext)){
		case 'apk':
			if(substr($file,0,4)==="PK\x03\x04"){
				$mime = Package::MIME_ANDROID;
				$platform = Package::PF_ANDROID;
			}
			break;
		case 'ipa':
			if(substr($file,0,4)==="PK\x03\x04"){
				$mime = Package::MIME_IOS;
				$platform = Package::PF_IOS;
			}
			break;
		}

		$temp_name = static::TEMP_DIR.Random::string(16).".$ext";
		S3::upload($temp_name,$file,$mime,'private');

		return array($temp_name,$platform);
	}
}

