<?php

/**
 * Row object for 'guest_pass' table
 */
class GuestPass extends mfwObject {
    const DB_CLASS = 'GuestPassDb';
    const SET_CLASS = 'GuestPassSet';

    public function getId(){
        return $this->value('id');
    }

    public function getMail(){
        return $this->value('mail');
    }

    public function getToken(){
        return $this->value('token');
    }

    public function getExpired(){
        return $this->value('expired');
    }
}

/**
 * Set of GuestPass objects.
 */
class GuestPassSet extends mfwObjectSet {
    public static function hypostatize(Array $row=array())
    {
        return new GuestPass($row);
    }
    protected function unsetCache($id)
    {
        parent::unsetCache($id);
    }
}

/**
 * database accessor for 'guest_pass' table.
 */
class GuestPassDb extends mfwObjectDb {
    const TABLE_NAME = 'guest_pass';
    const SET_CLASS = 'GuestPassSet';


    /**
     * @param integer $app_id
     * @param integer $package_id
     * @param string $mail
     * @param string $token
     * @param datetime $expired
     * @param PDO|null $con
     * @return GuestPass
     */
    public static function insertNewGuestPass(Package $package, User $user, $token, $expired, PDO $con=null)
    {
        $row = array(
            'app_id' => $package->getAppId(),
            'package_id' => $package->getId(),
            'mail' => $user->getMail(),
            'token' => $token,
            'expired' => $expired,
            'created' => date("Y-m-d H:i:s"),
        );
        $guest_pass = new GuestPass($row);
        $guest_pass->insert($con);
        return $guest_pass;
    }
}
