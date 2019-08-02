<?php
require_once __DIR__.'/actions.php';

class editActions extends packageActions
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

	public function executeEdit()
	{
		$params = array(
			);
		return $this->build($params);
	}

	public function executeEdit_commit()
	{
		$title = mfwRequest::param('title');
		$description = mfwRequest::param('description');
		$tag_names = mfwRequest::param('tags');
		$protect = (bool)mfwRequest::param('protect');

		if(!$title){
			error_log(__METHOD__.'('.__LINE__."): bad request: $temp_name, $title");
			return $this->response(self::HTTP_400_BADREQUEST);
		}

		$con = mfwDBConnection::getPDO();
		$con->beginTransaction();
		try{
			$app = ApplicationDb::retrieveByPKForUpdate($this->app->getId(),$con);
			$tags = $app->getOrInsertTagsByName($tag_names,$con);

			$pkg = PackageDb::retrieveByPKForUpdate($this->package->getId(),$con);
			$pkg->updateInfo($title,$description,$protect,$tags);

			$con->commit();
		}
		catch(Exception $e){
			$con->rollback();
			throw $e;
		}

		return $this->redirect("/package?id={$this->package->getId()}");
	}

}

