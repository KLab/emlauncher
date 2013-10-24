<!DOCTYPE html>
<html lang="ja">
  <head>
    <meta charset="utf-8">
    <title>EM-Launcher</title>
    <link rel="stylesheet" href="<?=url('/css/common.css')?>" type="text/css">
  </head>
  <body>
    <div id="navbar">
      <div id="loginuser" <?=($this->login_user)?'':'style="visibility:visible"'?>>
        <div class="address">
          makiuchi-d@klab.com
        </div>
        <ul id="usermenu">
          <li><a href="">logout</a></li>
          <li><a href="">logout</a></li>
          <li><a href="">logout</a></li>
        </ul>
      </div>
      <h1><span class="em">EM</span><span class="launcher">Launcher</span></h1>
    </div>
    <?=$contents?>
  </body>
  <script type="text/javascript">
  document.getElementById('loginuser').onclick = function(){
    var e = document.getElementById('usermenu');
    if(e.style.visibility=='visible'){
      e.style.visibility = 'hidden';
    }
    else{
      e.style.visibility = 'visible';
    }
  };
  </script>
</html>