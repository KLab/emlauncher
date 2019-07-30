<!DOCTYPE html>
<html lang="ja">
  <head>
    <meta charset="utf-8">
    <title><?=(isset($page_title))?htmlspecialchars($page_title).' | ':''?><?=$title_prefix?>EMLauncher</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
<?php if(mfwServerEnv::getEnv()==='local'): ?>
    <link href="/bootstrap/bootswatch/spacelab/bootstrap.min.css" rel="stylesheet" media="screen">
    <script src="/jquery/jquery.js"></script>
    <script src="/bootstrap/3.0.0/js/bootstrap.min.js"></script>
    <link href="/font-awesome/4.0.1/css/font-awesome.min.css" rel="stylesheet">
<?php else: ?>
    <link href="//netdna.bootstrapcdn.com/bootswatch/3.0.0/spacelab/bootstrap.min.css" rel="stylesheet" media="screen">
    <script src="//code.jquery.com/jquery.js"></script>
    <script src="//netdna.bootstrapcdn.com/bootstrap/3.0.0/js/bootstrap.min.js"></script>
    <link rel="stylesheet" href="//maxcdn.bootstrapcdn.com/font-awesome/4.3.0/css/font-awesome.min.css">
<?php endif ?>
    <link rel="stylesheet" href="<?=url('/css/customize.2.css')?>" type="text/css">
    <link rel="apple-touch-icon" href="<?=url('/apple-touch-icon.png')?>">
    <link rel="shortcut icon" href="<?=url('/favicon.ico')?>">
  </head>
  <body>

    <div class="navbar navbar-inverse" role="navigation">
      <div class="navbar-header">
        <button type="button" class="navbar-toggle" data-toggle="collapse" data-target=".navbar-ex1-collapse">
          <span class="sr-only">Toggle navigation</span>
          <span class="icon-bar"></span>
          <span class="icon-bar"></span>
          <span class="icon-bar"></span>
        </button>
        <a href="<?=url('/')?>" class="navbar-brand"><?=$title_prefix?><span>EMLauncher</span></a>
      </div>
      <div class="collapse navbar-collapse navbar-ex1-collapse">
<?php if($login_user): ?>
        <ul class="nav navbar-nav">
          <li><a href="<?=url('/')?>">Top</a></li>
          <li class="dropdown">
            <a hfer="#" class="dropdown-toggle" data-toggle="dropdown">My apps <b class="caret"></b></a>
            <ul class="dropdown-menu">
              <li><a href="<?=url('/myapps/installed')?>">Installed Apps</a></li>
              <li><a href="<?=url('/myapps/own')?>">Own Apps</a></li>
            </ul>
          </li>
          <li><a href="<?=url('/doc/api')?>">API doc</a></li>
        </ul>

        <ul class="nav navbar-nav navbar-right">
          <li class="dropdown">
            <a href="#" class="dropdown-toggle" data-toggle="dropdown"><?=htmlspecialchars($login_user->getMail())?> <b class="caret"></b></a>
            <ul class="dropdown-menu">
              <li><a href="/guestpass/list">GuestPass history</a></li>
              <li><a href="<?=url('/logout')?>">Logout</a></li>
            </ul>
          </li>
        </ul>
<?php else: ?>
        <ul class="nav navbar-nav navbar-right">
          <li><a href="<?=url('/login')?>">Login</a></li>
        </ul>
<?php endif ?>
      </div>
    </div>

    <div id="<?="{$module}-{$action}"?>" class="container">
      <?=$contents?>
    </div>
  </body>
</html>
