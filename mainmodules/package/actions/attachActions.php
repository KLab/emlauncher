<?php
require_once __DIR__.'/actions.php';

class attachActions extends packageActions
{
	public function initialize()
	{
		if(($err=parent::initialize())){
			return $err;
		}
		if($this->getAction()!='attach_download' && !$this->app->isOwner($this->login_user)){
			return $this->buildErrorPage(
				'Permission Denied',array(self::HTTP_403_FORBIDDEN));
		}
		return null;
   }

	public function executeAttach(){
		$file = mfwRequest::param('file');

		$type = AttachedFile::getTypeFromExt(pathinfo($file['name'],PATHINFO_EXTENSION));

		$con = mfwDBConnection::getPDO();
		$con->beginTransaction();
		try{
			$app = ApplicationDb::retrieveByPKForUpdate($this->app->getId());

			$attached = AttachedFileDb::insertNewAttachedFile(
				$this->package,$file['name'],$file['size'],$type,$con);

			$attached->uploadFile($file['tmp_name'],$file['type']);

			$con->commit();
		}
		catch(Exception $e){
			$con->rollback();
			throw $e;
		}

		return $this->redirect("/package?id={$this->package->getId()}");
	}
}
