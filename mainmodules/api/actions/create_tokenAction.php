<?php
require_once __DIR__.'/actions.php';
require_once APP_ROOT.'/model/Application.php';
require_once APP_ROOT.'/model/Package.php';

class create_tokenAction extends apiActions
{
    const INSTALL_TOKEN_PREFIX = 'pkg_install_token_';

    public function executeCreate_token()
    {
        try{
            $pkg_id      = mfwRequest::param('id');
            $mail        = mfwRequest::param('mail');
            $expire_hour = mfwRequest::param('expire_hour');

            $app = $this->app;

            // id check
            $pkg = PackageDb::retrieveByPK($pkg_id);
            if(!$pkg || $app->getId()!==$pkg->getAppId()){
                return $this->jsonResponse(
                    self::HTTP_400_BADREQUEST,
                    array('error'=>'Invalid package id'));
            }
            // mail check
            $owner_mails = $app->getOwners()->getMailArray();
            if ( ! in_array($mail, $owner_mails)){
                return $this->jsonResponse(
                    self::HTTP_400_BADREQUEST,
                    array('error'=>'Invalid mail address'));
            }

            // create install token
            $expire_hour = empty($expire_hour) ? 1 :  $expire_hour;
            $token_expire = sprintf('+%s hours', $expire_hour);

            $expire_time = strtotime($token_expire);
            $mc_expire = $expire_time - time();

            $tokendata = array(
                'mail' => $mail,
                'package_id' => $pkg_id,
                'expire' => date('Y-m-d H:i:s',$expire_time),
                );
            $token = Random::string(32);
            mfwMemcache::set(self::INSTALL_TOKEN_PREFIX.$token,json_encode($tokendata),$mc_expire);

            apache_log('token',$token);
            apache_log('token_data',$tokendata);

            $ret = self::makePackageArray($pkg);
            $ret['install_url'] = mfwRequest::makeURL("/package/install?token={$token}");

        }
        catch(Exception $e){
            error_log(__METHOD__.'('.__LINE__.'): '.get_class($e).":{$e->getMessage()}");
            return $this->jsonResponse(
                self::HTTP_500_INTERNALSERVERERROR,
                array('error'=>$e->getMessage(),'exception'=>get_class($e)));
        }

        apache_log('app_id',$app->getId());

        return $this->jsonResponse(self::HTTP_200_OK,$ret);
    }

}

