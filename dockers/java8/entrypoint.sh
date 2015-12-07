#!/bin/bash

cd /minecraft/data

if [ ! -e /minecraft/data/eula.txt ]; then
    if [ "$EULA" != "" ]; then
        echo "eula=true" > eula.txt
    else
        echo ""
        echo "Please accept the Minecraft EULA at"
        echo "  https://account.mojang.com/documents/minecraft_eula"
        echo "by adding the following immediately after 'docker run':"
        echo "  -e EULA=TRUE"
        echo ""
        exit 1
    fi
fi

case $VERSION in
    LATEST)
        VERSION=`wget -O - https://s3.amazonaws.com/Minecraft.Download/versions/versions.json | jsawk -n 'out(this.latest.release)'`
    ;;

    SNAPSHOT)
        VERSION=`wget -O - https://s3.amazonaws.com/Minecraft.Download/versions/versions.json | jsawk -n 'out(this.latest.snapshot)'`
    ;;

    *)
        VERSION=$VERSION
    ;;
esac

if [ "$FORGE_VERSION" == "" ]; then
    SERVER="minecraft_server.$VERSION.jar"
    [ ! -e $SERVER ] && wget -q https://s3.amazonaws.com/Minecraft.Download/versions/$VERSION/$SERVER
else
    case $VERSION in
        1.8.9 | 1.7.*)
            norm=$VERSION
        ;;

        *)
            norm=`echo $VERSION | sed 's/^\([0-9]\+\.[0-9]\+\).*/\1/'`
        ;;
    esac

  	case $FORGE_VERSION in
  	    RECOMMENDED)
  		    FORGE_VERSION=`wget -O - http://files.minecraftforge.net/maven/net/minecraftforge/forge/promotions_slim.json | jsawk -n "out(this.promos['$norm-recommended'])"`
  	    ;;
  	esac

    sorted=$((echo $FORGE_VERSION; echo 10.13.2.1300) | sort -V | head -1)
    if [[ $norm == '1.7.10' && $sorted == '10.13.2.1300' ]]; then
        normForgeVersion="$norm-$FORGE_VERSION-$norm"
    else
        normForgeVersion="$norm-$FORGE_VERSION"
    fi

    FORGE_INSTALLER="forge-$normForgeVersion-installer.jar"
    SERVER="forge-$normForgeVersion-universal.jar"

    if [ ! -e $SERVER ]; then
      wget -q http://files.minecraftforge.net/maven/net/minecraftforge/forge/$normForgeVersion/$FORGE_INSTALLER
      java -jar $FORGE_INSTALLER --installServer
    fi
fi

if [ ! -e server.properties ]; then
    [ -n "$WHITE_LIST" ] && echo "white-list=true" >> /minecraft/data/server.properties

    [ -n "$MOTD" ] && echo "motd=$MOTD" >> /minecraft/data/server.properties

    [ -n "$LEVEL" ] && echo "level-name=$LEVEL" >> /minecraft/data/server.properties

    [ -n "$SEED" ] && echo "level-seed=$SEED" >> /minecraft/data/server.properties

    [ -n "$PVP" ] && echo "pvp=$PVP" >> /minecraft/data/server.properties

    if [ -n "$LEVEL_TYPE" ]; then
        LEVEL_TYPE=${LEVEL_TYPE^^}
        case $LEVEL_TYPE in
            DEFAULT|FLAT|LARGEBIOMES|AMPLIFIED|CUSTOMIZED)
                echo "level-type=$LEVEL_TYPE" >> /minecraft/data/server.properties
            ;;

            *)
                echo "level-type=DEFAULT" >> /minecraft/data/server.properties
	        ;;
        esac
    fi

    [ -n "$GENERATOR_SETTINGS" ] && echo "generator-settings=$GENERATOR_SETTINGS" >> /minecraft/data/server.properties

    if [ -n "$DIFFICULTY" ]; then
        case $DIFFICULTY in
            peaceful)
                DIFFICULTY=0
            ;;

            easy)
                DIFFICULTY=1
            ;;

            normal)
                DIFFICULTY=2
            ;;

            hard)
                DIFFICULTY=3
            ;;

            *)
                DIFFICULTY=2
            ;;
        esac
        echo "difficulty=$DIFFICULTY" >> /minecraft/data/server.properties
    fi

    if [ -n "$MODE" ]; then
        case ${MODE,,?} in
            0|1|2|3)
            ;;

            s*)
                MODE=0
            ;;

            c*)
                MODE=1
            ;;

            *)
                MODE=0
            ;;
        esac
        echo "gamemode=$MODE" >> /minecraft/data/server.properties
    fi
fi


[ -n "$OPS" -a ! -e ops.txt.converted ] &&  echo $OPS | awk -v RS=, '{print}' >> ops.txt

[ -n "$WHITE_LIST" -a ! -e white-list.txt.converted ] && echo $WHITE_LIST | awk -v RS=, '{print}' >> white-list.txt

exec java -jar $SERVER $@
