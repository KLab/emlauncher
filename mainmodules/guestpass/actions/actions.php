<?php
require_once APP_ROOT.'/model/Application.php';
require_once APP_ROOT.'/model/Package.php';
require_once APP_ROOT.'/model/GuestPass.php';
require_once APP_ROOT . '/model/GuestPassLog.php';

class guestpassActions extends MainActions
{
    public function executeList()
    {
        $params = array(
        );
        return $this->build($params);
    }
}
