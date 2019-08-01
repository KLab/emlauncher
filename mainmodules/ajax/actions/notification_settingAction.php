<?php
require_once __DIR__.'/actions.php';

class notification_settingAction extends ajaxActions
{
	public function executeNotification_setting()
	{
		try{
			$app_id = (int)mfwRequest::param('id');
			$notify = (bool)mfwRequest::param('value',false);
			apache_log('app_id',$app_id);
			apache_log('value',$notify);

			$instapp = InstallLog::getInstallApp($this->login_user,$app_id);
			if(!$instapp){
				return $this->jsonResponse(
					self::HTTP_404_NOTFOUND,
					array('error'=>'installed application not found.'));
			}

			$instapp->updateNotifySetting($notify);
		}
		catch(Exception $e){
			error_log(__METHOD__.'('.__LINE__.'): '.get_class($e).":{$e->getMessage()}");
			return $this->jsonResponse(
				self::HTTP_500_INTERNALSERVERERROR,
				array('error'=>$e->getMessage(),'exception'=>get_class($e)));
		}

		return $this->jsonResponse(
			self::HTTP_200_OK,
			$instapp->toArray());
	}
}

