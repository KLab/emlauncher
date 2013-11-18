<?php
require_once __DIR__.'/actions.php';
require_once APP_ROOT.'/model/Package.php';
require_once APP_ROOT.'/model/IPAFile.php';

class upload_package_temporaryAction extends apiActions
{
	public function executeUpload_package_temporary()
	{
		try{
			$file_info = mfwRequest::param('file');
			if(!$file_info || $file_info['error']!=UPLOAD_ERR_OK){
				throw new InvalidArgumentException("upload files error: {$file_info['error']}");
			}
			$file_name = $file_info['name'];
			$file_path = $file_info['tmp_name'];
			$file_type = $file_info['type'];

			$file = file_get_contents($file_path);

			list($temp_name,$platform) = PackageDb::uploadTemporary($file_name,$file,$file_type);

			$ios_identifier = null;
			if($platform===Package::PF_IOS){
				$plist = IPAFile::parseInfoPlist($file_path);
				$ios_identifier = $plist['CFBundleIdentifier'];
			}
		}
		catch(Exception $e){
			error_log(__METHOD__.": {$e->getMessage()}");
			return $this->jsonResponse(
				self::HTTP_500_INTERNALSERVERERROR,
				array('error'=>$e->getMessage()));
		}
		return $this->jsonResponse(
			self::HTTP_200_OK,
			array(
				'file_name' => $file_name,
				'temp_name' => $temp_name,
				'platform' => $platform,
				'ios_identifier' => $ios_identifier,
				));
	}
}
