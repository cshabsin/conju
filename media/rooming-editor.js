var RoomingConfig = {
    desiredMask: 0,
    acceptableMask: 0
};

function combineProperties(existing, next) {
    var desired = (existing | next) & RoomingConfig.desiredMask;
    var acceptable = existing & next & RoomingConfig.acceptableMask;
    return desired + acceptable;
}

function propertiesMatch(roomProperties, peopleProperties) {
    var desiredMask = RoomingConfig.desiredMask;
    var paddingMask = 511; //TODO: generate this programmatically

    var bitwiseResult = (roomProperties & peopleProperties) |
        (desiredMask & paddingMask & ~peopleProperties) |
        (~desiredMask & paddingMask & ~roomProperties);

    return (~bitwiseResult & paddingMask) == 0;
}

function dropFromEvent(event) {
    event.preventDefault();
    drop(event.dataTransfer.getData('text'), $(event.target));
}

function drop(data, target) {
    var group = document.getElementById(data);
    $(target).append(group);

    var guestKeys = $(".guestKey", $(group));
    for (var i = 0; i < guestKeys.length; i++) {
        $("[name='roomingSlot_" + $(guestKeys[i]).text() + "']").val(target.attr("id"));
    }

    var properties = $(".guestProperties", target);
    // Checking if properties exist to avoid errors
    if (properties.length === 0) return;

    var totalProperties = parseInt($(properties[0]).text())
    if (properties.length > 1) {
        for (var i = 1; i < properties.length; i++) {
            totalProperties = combineProperties(totalProperties, parseInt($(properties[i]).text()));
        }
    }

    // Check if room properties element exists
    var roomPropsElem = $(".roomProperties", $(target));
    if (roomPropsElem.length > 0) {
        var roomProperties = parseInt(roomPropsElem.text());
        if (propertiesMatch(roomProperties, totalProperties)) {
            $(group).removeClass("constraintsNotMet");
        } else {
            $(group).addClass("constraintsNotMet");
        }
    }
}

function allowDrop(ev) {
    ev.preventDefault();
}

function drag(ev) {
    ev.dataTransfer.setData("text", ev.target.id);
}

function recordRoom(ev, entityId, isGuest) {
    var room = $($(ev.target)[0]).parent()[0].id;
    if (isGuest) {
        $('[name=room_assignment_guest' + entityId + ']').val(room);
    } else {
        $("#inv" + entityId + " .guest").each(function(index, g) {
            var guestId = parseInt($(g).attr("id").substring(5));
            $('[name=room_assignment_guest' + guestId + ']').val(room);
        });
    }
}

function explodeInvitee(groupId) {
    $("#" + groupId).removeClass(); //.addClass("guestContainer");
    $("#" + groupId + " .exploder").remove();
    $("#" + groupId).attr("draggable", "false").attr("ondragstart", "").attr("ondragend", "");
    $("#" + groupId + " .guest").attr("draggable", "true").attr("ondragstart", "drag(event)");
    $("#" + groupId + " .guest").addClass("groupContainer")
}

function initRoomingEditor(invitationsToExplode, bookingInfos) {
    if (invitationsToExplode) {
        invitationsToExplode.forEach(function(id) {
            var parent = $("#guest_" + id).parent();
            $(".exploder", parent).click();
        });
    }

    if (bookingInfos) {
        bookingInfos.forEach(function(info) {
            info.roommates.forEach(function(personKey) {
                var group = $("#guest_" + personKey).closest(".groupContainer");
                var target = $("#" + info.roomString);
                target.append(group);
                // The original code called drop logic, which sets hidden fields and checks constraints
                drop($(group).attr("id"), target);
            });
        });
    }

    // binding with jquery
    $(".floorplanLink").bind("mouseover",
        function() {
            var parent = $(this).parent();
            parent.find("img").css("opacity", 1.0);
            parent.find("div").css("opacity", .3);
            parent.find(".roomLabel").css("visibility", "hidden");
        });
    $(".floorplanLink").bind("mouseout",
        function() {
            var parent = $(this).parent();
            parent.find("img").css("opacity", .5);
            parent.find("div").css("opacity", 1);
            parent.find(".roomLabel").css("visibility", "visible");
        });
}
