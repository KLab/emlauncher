<?php
require_once __DIR__.'/actions.php';
require_once APP_ROOT.'/model/Package.php';

class uploadActions extends appActions
{
	public function initialize()
	{
		if(($err=parent::initialize())){
			return $err;
		}
		if(!$this->app->isOwner($this->login_user)){
			return $this->buildErrorPage(
				'Permission Denied',array(self::HTTP_403_FORBIDDEN));
		}
		return null;
	}

	public function executeUpload()
	{
		$params = array(
			);
		return $this->build($params);
	}

	public function executeUpload_post()
	{
		$temp_name = mfwRequest::param('temp_name');
		$platform = mfwRequest::param('platform');
		$title = mfwRequest::param('title');
		$description = mfwRequest::param('description');
		$tag_names = mfwRequest::param('tags');
		$ios_identifier = mfwRequest::param('ios_identifier');

		if(!$temp_name || !$title){
			error_log(__METHOD__.": bad request: $temp_name, $title");
			return $this->response(self::HTTP_400_BADREQUEST);
		}
		$ext = pathinfo($temp_name,PATHINFO_EXTENSION);

		$con = mfwDBConnection::getPDO();
		$con->beginTransaction();
		try{
			$app = ApplicationDb::retrieveByPKForUpdate($this->app->getId(),$con);

			$tags = $app->getTagsByName($tag_names,$con);

			$pkg = PackageDb::insertNewPackage(
				$this->app->getId(),$platform,$ext,$title,$description,$ios_identifier,$tags,$con);

			$pkg->renameTempFile($temp_name);

			$app->updateLastUpload($con);

			$con->commit();
		}
		catch(Exception $e){
			$con->rollback();
			throw $e;
		}

		// todo: notification

		return $this->redirect("/package?id={$pkg->getId()}");
	}

}

