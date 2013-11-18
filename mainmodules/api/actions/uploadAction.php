<?php
require_once __DIR__.'/actions.php';
require_once APP_ROOT.'/model/Application.php';
require_once APP_ROOT.'/model/Package.php';

class uploadAction extends apiActions
{
	public function executeUpload()
	{
		$api_key = mfwRequest::param('api_key');
		$file = mfwRequest::param('file');
		$title = mfwRequest::param('title');
		$description = mfwRequest::param('description');
		$notify = mfwRequest::param('notify');
		$tag_names = explode(',',mfwRequest::param('tags'));
		if(!$api_key||!$file||!$title){
			return $this->jsonResponse(
				self::HTTP_400_BADREQUEST,
				array('error'=>'A required field is not present.'));
		}

		$app = ApplicationDb::selectByApiKey($api_key);
		if(!$app){
			return $this->jsonResponse(
				self::HTTP_400_BADREQUEST,
				array('error'=>'Invalid api_key'));
		}

		$con = mfwDBConnection::getPDO();
		$con->beginTransaction();
		try{
			$app = ApplicationDb::retrieveByPKForUpdate($app->getId());

			$tags = $app->getTagsByName($tag_names,$con);

			$pkg = PackageDb::uploadAndInsertNewPackage(
				$app->getId(),$file['name'],$file['tmp_name'],$file['type'],
				$title,$description,$tags,$con);

			$con->commit();
		}
		catch(Exception $e){
			error_log(__METHOD__."$e->getMessage()");
			return $this->jsonResponse(
				self::HTTP_500_INTERNALSERVERERROR,
				array('error'=>$e->getMessage()));
		}

		return array(array(self::HTTP_200_OK),json_encode($pkg));
	}

}

