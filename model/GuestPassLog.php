<?php

/**
 */
class GuestPassLog {

    public static function Logging(GuestPass $guest_pass, mfwUserAgent $ua, $ip_address, $con=null)
    {
        $now = date('Y-m-d H:i:s');
        $sql = 'INSERT INTO guestpass_log'
            . ' (guest_pass_id,user_agent,ip_address, installed)'
            . ' VALUES (:guest_pass_id,:user_agent,:ip_address, :installed)';
        $bind = array(
            ':guest_pass_id' => $guest_pass->getId(),
            ':user_agent' => $ua->getString(),
            ':ip_address' => $ip_address,
            ':installed' => $now,
        );
        mfwDBIBase::query($sql,$bind,$con);
    }

    public static function selectCountByGuestPassId($guest_pass_id)
    {
        $sql = "SELECT COUNT(*) FROM guestpass_log WHERE guest_pass_id = ?";
        return (int)mfwDBIBase::getOne($sql,array($guest_pass_id));
    }
}
