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

    public function getPackageId(){
        return $this->value('package_id');
    }

    public function getMail(){
        return $this->value('mail');
    }

    public function getToken(){
        return $this->value('token');
    }

    public function getCreated(){
        return $this->value('created');
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
     * @param Package $package
     * @param User $user
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

    /**
     * @param string $token
     * @param PDO|null $con
     * @return GuestPass|null
     */
    public static function selectByToken($token, PDO $con=null)
    {
        $query = 'WHERE token = :token LIMIT 1';
        $bind = array(':token' => $token);
        return static::selectOne($query,$bind);
    }

    public static function selectByPackageId($pakcage_id, $is_active = true)
    {
        $sql = 'SELECT * FROM guest_pass WHERE package_id = ?';
        if ($is_active) {
            $sql .=" AND expired > '".date("Y-m-d H:i:s")."'";
        }
        $rows = mfwDBIBase::getAll($sql,array($pakcage_id));
        return new GuestPassSet($rows);
    }
}
