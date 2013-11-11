<?php
require_once __DIR__.'/actions.php';
require_once APP_ROOT.'/model/S3.php';

class upload_package_temporaryAction extends apiActions
{
	public function executeUpload_package_temporary()
	{
		try{
			$file_name = mfwRequest::param('name');
			$file = mfwRequest::body();
			$temp_name = 'temp_name.apk';
			$type = 'Android';
			sleep(1);
		}
		catch(Exception $e){
			error_log(__METHOD__." {$e->getMessage()}");
			return $this->jsonResponse(
				self::HTTP_500_INTERNALSERVERERROR,
				array('message'=>$e->getMessage()));
		}
		return $this->jsonResponse(
			self::HTTP_200_OK,
			array(
				'file_name' => $file_name,
				'temp_name' => $temp_name,
				'type' => $type,
				));
	}
}
