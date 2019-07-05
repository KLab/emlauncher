<?php
require_once __DIR__.'/actions.php';
require_once APP_ROOT.'/model/AttachedFile.php';

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

	public function executeAttach_download(){
		$attached_id = mfwRequest::param('attached_id');

		$attached = AttachedFileDb::retrieveByPK($attached_id);
		if(!$attached){
			return $this->buildErrorPage('Not Found', array(self::HTTP_404_NOTFOUND));
		}
		if($attached->getPackageId() != $this->package->getId()){
			return $this->buildErrorPage('Not Found', array(self::HTTP_404_NOTFOUND));
		}

		$url = $attached->getFileUrl("+60 min");

		apache_log('app_id',$this->app->getId());
		apache_log('pkg_id',$this->package->getId());
		apache_log('attached_id',$attached_id);

		return $this->redirect($url);
	}
}
