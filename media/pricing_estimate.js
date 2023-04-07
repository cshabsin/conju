
const babyLimitDate = new Date(2019, 6, 11);
const kidLimitDate = new Date(2011, 6, 11);


function computeCost() {
  var rsvps = $('select[name="rsvp"]').map(function() {return $(this).val()}).toArray();
  var totalAttendees = 0;
  for (var x = 0; x < rsvps.length; x++) {
    if (rsvps[x] == "2" || rsvps[x] == "3") totalAttendees++;
  }
  
  if (totalAttendees == 1 && $('select[name="housingPreference"]').val() != "1") {
    $(".cost0Roommates").text("$" + computeCostWithRoommates(0));
    $(".cost1Roommates").text("$" + computeCostWithRoommates(1));
    $(".cost2Roommates").text("$" + computeCostWithRoommates(2));
    $(".cost3Roommates").text("$" + computeCostWithRoommates(3)); 
    $(".onlyCost").hide();
    $(".roommateCosts").show();

  } else {
    $(".onlyCost").text("$" + computeCostWithRoommates(0));
    $(".roommateCosts").hide();
    $(".onlyCost").show();
  }
}

function computeCostWithRoommates(additionalPeople) {

  var numberOfNights = 0;
  if (anyoneMeetsCriteria(attending)) {
    numberOfNights = 2;
  }
  if (anyoneMeetsCriteria(thursday)) {
    numberOfNights = 3;
  }


  var roomRate = 175;
  if ($('input[name="housingPreferenceBooleans"][value="7"]').is(":checked")) roomRate = 230;

  var rsvps = $('select[name="rsvp"]').map(function() {return $(this).val()}).toArray();
  var totalAttendees = 0;
  for (var x = 0; x < rsvps.length; x++) {
    if (rsvps[x] == "2" || rsvps[x] == "3") totalAttendees++;
  }

  // if only 1 person and no roommates, set room rate to single room cost
  if (totalAttendees == 1) {
    if ($('select[name="housingPreference"]').val() == "1") {
      roomRate = 140;
    } else {
      roomRate /= (additionalPeople + 1);
    }
  }


  var lodgingCost = roomRate * numberOfNights;
  //console.log("lodging cost: " + lodgingCost);
  var foodCost = 0;

  var fridayLunchCost = 10;
  var adultFridayDinnerCost = 15; 
  var kidFridayDinnerCost = 10;
  var adultFoodCost = 140;
  var kidFoodCost = 63;
  var babyFoodCost = 0;
  
  const pricesByAgeAndNights = [
     [adultFridayDinnerCost + adultFoodCost, adultFridayDinnerCost + adultFoodCost + fridayLunchCost],
     [kidFridayDinnerCost + kidFoodCost, kidFridayDinnerCost + kidFoodCost + fridayLunchCost],
     [0, 0]
  ]

  var birthdates = $('input[name="Birthdate"]').map(function() {return $(this).val()}).toArray();

  for (var i=0; i < rsvps.length; i++) {

    var nightsIndex = -1;
    if (rsvps[i] == "2") nightsIndex = 0;
    else if (rsvps[i] == "3") nightsIndex = 1;
    else continue;

    var ageIndex = 0;
    const birthdateStr = birthdates[i];
    if (birthdateStr && birthdateStr.split("/").length == 3) {
      var pieces = birthdateStr.split("/").map(function(x) {return Number(x);})

      const date = new Date(pieces[2], pieces[0], pieces[1]);
      if (date > babyLimitDate) ageIndex = 2;
      else if (date > kidLimitDate) ageIndex = 1;
    }
    
    foodCost += pricesByAgeAndNights[ageIndex][nightsIndex];
  }

  //console.log("food cost: " + foodCost); 

  var incidentalsPerPersonPerNight = 5;
 
  var incidentalsCost = incidentalsPerPersonPerNight *  $('select[name="rsvp"] option:selected').map(function() {
    if ($( this ).val() == "3") return 3;
    if ($( this ).val() == "2") return 2;
  }).toArray().reduce((current, total) => current + total, 0)
      //console.log("incidentals cost: " + incidentalsCost);

  var totalCost = lodgingCost + foodCost + incidentalsCost;
  $(".estimatedCost").text("$" + totalCost)

      //console.log(totalCost);
  //alert(totalCost);
  return Math.ceil(totalCost);

}


