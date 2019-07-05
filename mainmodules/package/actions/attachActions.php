<?php
require_once __DIR__.'/actions.php';
require_once APP_ROOT.'/model/AttachedFile.php';
require_once APP_ROOT.'/model/Random.php';

class attachActions extends packageActions
{
	const SESKEY_TOKEN = 'attached_delete_token';

	protected $attached = null;

	public function initialize()
	{
		if(($err=parent::initialize())){
			return $err;
		}
		if($this->getAction()!='attach_download' && !$this->app->isOwner($this->login_user)){
			return $this->buildErrorPage(
				'Permission Denied',array(self::HTTP_403_FORBIDDEN));
		}
		if($this->getAction()!='attach'){
			$id = mfwRequest::param('attached_id');
			$this->attached = AttachedFileDb::retrieveByPK($id);
			if(!$this->attached || $this->attached->getPackageId()!=$this->package->getId()){
				return $this->buildErrorPage('Not Found',array(self::HTTP_404_NOTFOUND));
			}
		}
		return null;
   }

	public function executeAttach(){
		$file = mfwRequest::param('file');

		$type = AttachedFile::getTypeFromExt(pathinfo($file['name'],PATHINFO_EXTENSION));

		$con = mfwDBConnection::getPDO();
		$con->beginTransaction();
		try{
			$app = ApplicationDb::retrieveByPKForUpdate($this->app->getId(),$con);

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
		$url = $this->attached->getFileUrl("+60 min");

		apache_log('app_id',$this->app->getId());
		apache_log('pkg_id',$this->package->getId());
		apache_log('attached_id',$this->attached->getId());

		return $this->redirect($url);
	}

	public function executeAttach_delete_confirm(){
		$token	= Random::string(32);
		mfwSession::set(self::SESKEY_TOKEN,$token);

		$parma = array(
			'token' => $token,
			'attached' => $this->attached,
			);
		return $this->build($parma);
	}

	public function executeAttach_delete(){
		$token = mfwRequest::param('token',null,'POST');
		if($token!==mfwSession::get(self::SESKEY_TOKEN)){
			return $this->buildErrorPage(
				'Bad Request',array(self::HTTP_400_BADREQUEST));
		}
		mfwSession::clear(self::SESKEY_TOKEN);

		apache_log('app_id',$this->app->getId());
		apache_log('pkg_id',$this->package->getId());
		apache_log('attached_id',$this->attached->getId());

		$con = mfwDBConnection::getPDO();
		$con->beginTransaction();
		try{
			$this->app = ApplicationDb::retrieveByPKForUpdate($this->app->getId(),$con);
			$this->attached->delete($con);
			$con->commit();
		}
		catch(Exception $e){
			$con->rollback();
			error_log(__METHOD__.'('.__LINE__.'): '.get_class($e).":{$e->getMessage()}");
			throw $e;
		}

		try{
			$this->attached->deleteFile();
		}
		catch(Exception $e){
			// S3から削除出来なくてもDBからは消えているので許容する
			error_log(__METHOD__.'('.__LINE__.'): '.get_class($e).":{$e->getMessage()}");
		}

		return $this->redirect("/package?id={$this->package->getId()}");
	}
}
