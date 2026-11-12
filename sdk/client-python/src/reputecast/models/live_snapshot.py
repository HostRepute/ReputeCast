from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.live_snapshot_state import LiveSnapshotState
from ..models.live_snapshot_status import LiveSnapshotStatus
from ..models.live_snapshot_threshold_scope import LiveSnapshotThresholdScope
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.live_ab import LiveAB
  from ..models.live_canary import LiveCanary
  from ..models.live_cost_by_route import LiveCostByRoute
  from ..models.live_details import LiveDetails
  from ..models.live_event import LiveEvent
  from ..models.live_excluded import LiveExcluded
  from ..models.live_failover import LiveFailover
  from ..models.live_helpers import LiveHelpers
  from ..models.live_in_doubt import LiveInDoubt
  from ..models.live_incident import LiveIncident
  from ..models.live_limits import LiveLimits
  from ..models.live_mailbox import LiveMailbox
  from ..models.live_options import LiveOptions
  from ..models.live_postmaster import LivePostmaster
  from ..models.live_provider import LiveProvider
  from ..models.live_route_share import LiveRouteShare
  from ..models.live_share import LiveShare
  from ..models.live_threshold import LiveThreshold
  from ..models.live_throttle import LiveThrottle
  from ..models.live_totals import LiveTotals
  from ..models.live_watch import LiveWatch





T = TypeVar("T", bound="LiveSnapshot")



@_attrs_define
class LiveSnapshot:
    """ 
        Attributes:
            approved_by (str):
            at (datetime.datetime):
            campaign_id (str):
            continuing (list[str]):
            cost (LiveCostByRoute):
            delivered_mix (list[LiveShare]):
            details (LiveDetails):
            error (str):
            events (list[LiveEvent]):
            events_via (str):
            excluded (LiveExcluded):
            failovers (list[LiveFailover]):
            held (int):
            incidents (list[LiveIncident]):
            limits (LiveLimits):
            mailboxes (list[LiveMailbox]):
            mailboxes_held (list[str]):
            name (str):
            options (LiveOptions):
            pause_reason (str):
            pause_scope_hint (str):
            paused_by (str):
            percent (float):
            providers (list[LiveProvider]):
            providers_held (list[str]):
            queued (int):
            rate (float): Messages per second now
            recipients (int):
            resumed (bool):
            retrying (int):
            routes (list[LiveRouteShare]):
            scheduled_by (str):
            sent (int):
            shared_streams (list[str]):
            started_by (str):
            state (LiveSnapshotState):
            status (LiveSnapshotStatus):
            threshold_scope (LiveSnapshotThresholdScope):
            thresholds (list[LiveThreshold]):
            throttles (list[LiveThrottle]):
            totals (LiveTotals):
            unsent (int):
            version (int): Milliseconds since the epoch; the SSE event id
            waiting (int):
            ab (LiveAB | Unset):
            approved_at (datetime.datetime | Unset):
            canary (LiveCanary | Unset):
            completed_at (datetime.datetime | Unset):
            deferral_window (int | Unset):
            duration_seconds (int | Unset):
            eta_seconds (int | Unset):
            finishes_at (datetime.datetime | Unset):
            helpers (LiveHelpers | Unset):
            in_doubt (LiveInDoubt | Unset):
            incident (LiveIncident | Unset):
            last_event_at (datetime.datetime | Unset):
            paused_at (datetime.datetime | Unset):
            postmaster (LivePostmaster | Unset):
            prepared_at (datetime.datetime | Unset):
            scheduled_at (datetime.datetime | Unset):
            started_at (datetime.datetime | Unset):
            watch (LiveWatch | Unset):
     """

    approved_by: str
    at: datetime.datetime
    campaign_id: str
    continuing: list[str]
    cost: LiveCostByRoute
    delivered_mix: list[LiveShare]
    details: LiveDetails
    error: str
    events: list[LiveEvent]
    events_via: str
    excluded: LiveExcluded
    failovers: list[LiveFailover]
    held: int
    incidents: list[LiveIncident]
    limits: LiveLimits
    mailboxes: list[LiveMailbox]
    mailboxes_held: list[str]
    name: str
    options: LiveOptions
    pause_reason: str
    pause_scope_hint: str
    paused_by: str
    percent: float
    providers: list[LiveProvider]
    providers_held: list[str]
    queued: int
    rate: float
    recipients: int
    resumed: bool
    retrying: int
    routes: list[LiveRouteShare]
    scheduled_by: str
    sent: int
    shared_streams: list[str]
    started_by: str
    state: LiveSnapshotState
    status: LiveSnapshotStatus
    threshold_scope: LiveSnapshotThresholdScope
    thresholds: list[LiveThreshold]
    throttles: list[LiveThrottle]
    totals: LiveTotals
    unsent: int
    version: int
    waiting: int
    ab: LiveAB | Unset = UNSET
    approved_at: datetime.datetime | Unset = UNSET
    canary: LiveCanary | Unset = UNSET
    completed_at: datetime.datetime | Unset = UNSET
    deferral_window: int | Unset = UNSET
    duration_seconds: int | Unset = UNSET
    eta_seconds: int | Unset = UNSET
    finishes_at: datetime.datetime | Unset = UNSET
    helpers: LiveHelpers | Unset = UNSET
    in_doubt: LiveInDoubt | Unset = UNSET
    incident: LiveIncident | Unset = UNSET
    last_event_at: datetime.datetime | Unset = UNSET
    paused_at: datetime.datetime | Unset = UNSET
    postmaster: LivePostmaster | Unset = UNSET
    prepared_at: datetime.datetime | Unset = UNSET
    scheduled_at: datetime.datetime | Unset = UNSET
    started_at: datetime.datetime | Unset = UNSET
    watch: LiveWatch | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.live_ab import LiveAB # noqa: PLC0415
        from ..models.live_canary import LiveCanary # noqa: PLC0415
        from ..models.live_cost_by_route import LiveCostByRoute # noqa: PLC0415
        from ..models.live_details import LiveDetails # noqa: PLC0415
        from ..models.live_event import LiveEvent # noqa: PLC0415
        from ..models.live_excluded import LiveExcluded # noqa: PLC0415
        from ..models.live_failover import LiveFailover # noqa: PLC0415
        from ..models.live_helpers import LiveHelpers # noqa: PLC0415
        from ..models.live_in_doubt import LiveInDoubt # noqa: PLC0415
        from ..models.live_incident import LiveIncident # noqa: PLC0415
        from ..models.live_limits import LiveLimits # noqa: PLC0415
        from ..models.live_mailbox import LiveMailbox # noqa: PLC0415
        from ..models.live_options import LiveOptions # noqa: PLC0415
        from ..models.live_postmaster import LivePostmaster # noqa: PLC0415
        from ..models.live_provider import LiveProvider # noqa: PLC0415
        from ..models.live_route_share import LiveRouteShare # noqa: PLC0415
        from ..models.live_share import LiveShare # noqa: PLC0415
        from ..models.live_threshold import LiveThreshold # noqa: PLC0415
        from ..models.live_throttle import LiveThrottle # noqa: PLC0415
        from ..models.live_totals import LiveTotals # noqa: PLC0415
        from ..models.live_watch import LiveWatch # noqa: PLC0415
        approved_by = self.approved_by

        at = self.at.isoformat()

        campaign_id = self.campaign_id

        continuing = self.continuing



        cost = self.cost.to_dict()

        delivered_mix = []
        for delivered_mix_item_data in self.delivered_mix:
            delivered_mix_item = delivered_mix_item_data.to_dict()
            delivered_mix.append(delivered_mix_item)



        details = self.details.to_dict()

        error = self.error

        events = []
        for events_item_data in self.events:
            events_item = events_item_data.to_dict()
            events.append(events_item)



        events_via = self.events_via

        excluded = self.excluded.to_dict()

        failovers = []
        for failovers_item_data in self.failovers:
            failovers_item = failovers_item_data.to_dict()
            failovers.append(failovers_item)



        held = self.held

        incidents = []
        for incidents_item_data in self.incidents:
            incidents_item = incidents_item_data.to_dict()
            incidents.append(incidents_item)



        limits = self.limits.to_dict()

        mailboxes = []
        for mailboxes_item_data in self.mailboxes:
            mailboxes_item = mailboxes_item_data.to_dict()
            mailboxes.append(mailboxes_item)



        mailboxes_held = self.mailboxes_held



        name = self.name

        options = self.options.to_dict()

        pause_reason = self.pause_reason

        pause_scope_hint = self.pause_scope_hint

        paused_by = self.paused_by

        percent = self.percent

        providers = []
        for providers_item_data in self.providers:
            providers_item = providers_item_data.to_dict()
            providers.append(providers_item)



        providers_held = self.providers_held



        queued = self.queued

        rate = self.rate

        recipients = self.recipients

        resumed = self.resumed

        retrying = self.retrying

        routes = []
        for routes_item_data in self.routes:
            routes_item = routes_item_data.to_dict()
            routes.append(routes_item)



        scheduled_by = self.scheduled_by

        sent = self.sent

        shared_streams = self.shared_streams



        started_by = self.started_by

        state = self.state.value

        status = self.status.value

        threshold_scope = self.threshold_scope.value

        thresholds = []
        for thresholds_item_data in self.thresholds:
            thresholds_item = thresholds_item_data.to_dict()
            thresholds.append(thresholds_item)



        throttles = []
        for throttles_item_data in self.throttles:
            throttles_item = throttles_item_data.to_dict()
            throttles.append(throttles_item)



        totals = self.totals.to_dict()

        unsent = self.unsent

        version = self.version

        waiting = self.waiting

        ab: dict[str, Any] | Unset = UNSET
        if not isinstance(self.ab, Unset):
            ab = self.ab.to_dict()

        approved_at: str | Unset = UNSET
        if not isinstance(self.approved_at, Unset):
            approved_at = self.approved_at.isoformat()

        canary: dict[str, Any] | Unset = UNSET
        if not isinstance(self.canary, Unset):
            canary = self.canary.to_dict()

        completed_at: str | Unset = UNSET
        if not isinstance(self.completed_at, Unset):
            completed_at = self.completed_at.isoformat()

        deferral_window = self.deferral_window

        duration_seconds = self.duration_seconds

        eta_seconds = self.eta_seconds

        finishes_at: str | Unset = UNSET
        if not isinstance(self.finishes_at, Unset):
            finishes_at = self.finishes_at.isoformat()

        helpers: dict[str, Any] | Unset = UNSET
        if not isinstance(self.helpers, Unset):
            helpers = self.helpers.to_dict()

        in_doubt: dict[str, Any] | Unset = UNSET
        if not isinstance(self.in_doubt, Unset):
            in_doubt = self.in_doubt.to_dict()

        incident: dict[str, Any] | Unset = UNSET
        if not isinstance(self.incident, Unset):
            incident = self.incident.to_dict()

        last_event_at: str | Unset = UNSET
        if not isinstance(self.last_event_at, Unset):
            last_event_at = self.last_event_at.isoformat()

        paused_at: str | Unset = UNSET
        if not isinstance(self.paused_at, Unset):
            paused_at = self.paused_at.isoformat()

        postmaster: dict[str, Any] | Unset = UNSET
        if not isinstance(self.postmaster, Unset):
            postmaster = self.postmaster.to_dict()

        prepared_at: str | Unset = UNSET
        if not isinstance(self.prepared_at, Unset):
            prepared_at = self.prepared_at.isoformat()

        scheduled_at: str | Unset = UNSET
        if not isinstance(self.scheduled_at, Unset):
            scheduled_at = self.scheduled_at.isoformat()

        started_at: str | Unset = UNSET
        if not isinstance(self.started_at, Unset):
            started_at = self.started_at.isoformat()

        watch: dict[str, Any] | Unset = UNSET
        if not isinstance(self.watch, Unset):
            watch = self.watch.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "approvedBy": approved_by,
            "at": at,
            "campaignId": campaign_id,
            "continuing": continuing,
            "cost": cost,
            "deliveredMix": delivered_mix,
            "details": details,
            "error": error,
            "events": events,
            "eventsVia": events_via,
            "excluded": excluded,
            "failovers": failovers,
            "held": held,
            "incidents": incidents,
            "limits": limits,
            "mailboxes": mailboxes,
            "mailboxesHeld": mailboxes_held,
            "name": name,
            "options": options,
            "pauseReason": pause_reason,
            "pauseScopeHint": pause_scope_hint,
            "pausedBy": paused_by,
            "percent": percent,
            "providers": providers,
            "providersHeld": providers_held,
            "queued": queued,
            "rate": rate,
            "recipients": recipients,
            "resumed": resumed,
            "retrying": retrying,
            "routes": routes,
            "scheduledBy": scheduled_by,
            "sent": sent,
            "sharedStreams": shared_streams,
            "startedBy": started_by,
            "state": state,
            "status": status,
            "thresholdScope": threshold_scope,
            "thresholds": thresholds,
            "throttles": throttles,
            "totals": totals,
            "unsent": unsent,
            "version": version,
            "waiting": waiting,
        })
        if ab is not UNSET:
            field_dict["ab"] = ab
        if approved_at is not UNSET:
            field_dict["approvedAt"] = approved_at
        if canary is not UNSET:
            field_dict["canary"] = canary
        if completed_at is not UNSET:
            field_dict["completedAt"] = completed_at
        if deferral_window is not UNSET:
            field_dict["deferralWindow"] = deferral_window
        if duration_seconds is not UNSET:
            field_dict["durationSeconds"] = duration_seconds
        if eta_seconds is not UNSET:
            field_dict["etaSeconds"] = eta_seconds
        if finishes_at is not UNSET:
            field_dict["finishesAt"] = finishes_at
        if helpers is not UNSET:
            field_dict["helpers"] = helpers
        if in_doubt is not UNSET:
            field_dict["inDoubt"] = in_doubt
        if incident is not UNSET:
            field_dict["incident"] = incident
        if last_event_at is not UNSET:
            field_dict["lastEventAt"] = last_event_at
        if paused_at is not UNSET:
            field_dict["pausedAt"] = paused_at
        if postmaster is not UNSET:
            field_dict["postmaster"] = postmaster
        if prepared_at is not UNSET:
            field_dict["preparedAt"] = prepared_at
        if scheduled_at is not UNSET:
            field_dict["scheduledAt"] = scheduled_at
        if started_at is not UNSET:
            field_dict["startedAt"] = started_at
        if watch is not UNSET:
            field_dict["watch"] = watch

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.live_ab import LiveAB # noqa: PLC0415
        from ..models.live_canary import LiveCanary # noqa: PLC0415
        from ..models.live_cost_by_route import LiveCostByRoute # noqa: PLC0415
        from ..models.live_details import LiveDetails # noqa: PLC0415
        from ..models.live_event import LiveEvent # noqa: PLC0415
        from ..models.live_excluded import LiveExcluded # noqa: PLC0415
        from ..models.live_failover import LiveFailover # noqa: PLC0415
        from ..models.live_helpers import LiveHelpers # noqa: PLC0415
        from ..models.live_in_doubt import LiveInDoubt # noqa: PLC0415
        from ..models.live_incident import LiveIncident # noqa: PLC0415
        from ..models.live_limits import LiveLimits # noqa: PLC0415
        from ..models.live_mailbox import LiveMailbox # noqa: PLC0415
        from ..models.live_options import LiveOptions # noqa: PLC0415
        from ..models.live_postmaster import LivePostmaster # noqa: PLC0415
        from ..models.live_provider import LiveProvider # noqa: PLC0415
        from ..models.live_route_share import LiveRouteShare # noqa: PLC0415
        from ..models.live_share import LiveShare # noqa: PLC0415
        from ..models.live_threshold import LiveThreshold # noqa: PLC0415
        from ..models.live_throttle import LiveThrottle # noqa: PLC0415
        from ..models.live_totals import LiveTotals # noqa: PLC0415
        from ..models.live_watch import LiveWatch # noqa: PLC0415
        d = dict(src_dict)
        approved_by = d.pop("approvedBy")

        at = datetime.datetime.fromisoformat(d.pop("at"))




        campaign_id = d.pop("campaignId")

        continuing = cast(list[str], d.pop("continuing"))


        cost = LiveCostByRoute.from_dict(d.pop("cost"))




        delivered_mix = []
        _delivered_mix = d.pop("deliveredMix")
        for delivered_mix_item_data in (_delivered_mix):
            delivered_mix_item = LiveShare.from_dict(delivered_mix_item_data)



            delivered_mix.append(delivered_mix_item)


        details = LiveDetails.from_dict(d.pop("details"))




        error = d.pop("error")

        events = []
        _events = d.pop("events")
        for events_item_data in (_events):
            events_item = LiveEvent.from_dict(events_item_data)



            events.append(events_item)


        events_via = d.pop("eventsVia")

        excluded = LiveExcluded.from_dict(d.pop("excluded"))




        failovers = []
        _failovers = d.pop("failovers")
        for failovers_item_data in (_failovers):
            failovers_item = LiveFailover.from_dict(failovers_item_data)



            failovers.append(failovers_item)


        held = d.pop("held")

        incidents = []
        _incidents = d.pop("incidents")
        for incidents_item_data in (_incidents):
            incidents_item = LiveIncident.from_dict(incidents_item_data)



            incidents.append(incidents_item)


        limits = LiveLimits.from_dict(d.pop("limits"))




        mailboxes = []
        _mailboxes = d.pop("mailboxes")
        for mailboxes_item_data in (_mailboxes):
            mailboxes_item = LiveMailbox.from_dict(mailboxes_item_data)



            mailboxes.append(mailboxes_item)


        mailboxes_held = cast(list[str], d.pop("mailboxesHeld"))


        name = d.pop("name")

        options = LiveOptions.from_dict(d.pop("options"))




        pause_reason = d.pop("pauseReason")

        pause_scope_hint = d.pop("pauseScopeHint")

        paused_by = d.pop("pausedBy")

        percent = d.pop("percent")

        providers = []
        _providers = d.pop("providers")
        for providers_item_data in (_providers):
            providers_item = LiveProvider.from_dict(providers_item_data)



            providers.append(providers_item)


        providers_held = cast(list[str], d.pop("providersHeld"))


        queued = d.pop("queued")

        rate = d.pop("rate")

        recipients = d.pop("recipients")

        resumed = d.pop("resumed")

        retrying = d.pop("retrying")

        routes = []
        _routes = d.pop("routes")
        for routes_item_data in (_routes):
            routes_item = LiveRouteShare.from_dict(routes_item_data)



            routes.append(routes_item)


        scheduled_by = d.pop("scheduledBy")

        sent = d.pop("sent")

        shared_streams = cast(list[str], d.pop("sharedStreams"))


        started_by = d.pop("startedBy")

        state = LiveSnapshotState(d.pop("state"))




        status = LiveSnapshotStatus(d.pop("status"))




        threshold_scope = LiveSnapshotThresholdScope(d.pop("thresholdScope"))




        thresholds = []
        _thresholds = d.pop("thresholds")
        for thresholds_item_data in (_thresholds):
            thresholds_item = LiveThreshold.from_dict(thresholds_item_data)



            thresholds.append(thresholds_item)


        throttles = []
        _throttles = d.pop("throttles")
        for throttles_item_data in (_throttles):
            throttles_item = LiveThrottle.from_dict(throttles_item_data)



            throttles.append(throttles_item)


        totals = LiveTotals.from_dict(d.pop("totals"))




        unsent = d.pop("unsent")

        version = d.pop("version")

        waiting = d.pop("waiting")

        _ab = d.pop("ab", UNSET)
        ab: LiveAB | Unset
        if isinstance(_ab,  Unset):
            ab = UNSET
        else:
            ab = LiveAB.from_dict(_ab)




        _approved_at = d.pop("approvedAt", UNSET)
        approved_at: datetime.datetime | Unset
        if isinstance(_approved_at,  Unset):
            approved_at = UNSET
        else:
            approved_at = datetime.datetime.fromisoformat(_approved_at)




        _canary = d.pop("canary", UNSET)
        canary: LiveCanary | Unset
        if isinstance(_canary,  Unset):
            canary = UNSET
        else:
            canary = LiveCanary.from_dict(_canary)




        _completed_at = d.pop("completedAt", UNSET)
        completed_at: datetime.datetime | Unset
        if isinstance(_completed_at,  Unset):
            completed_at = UNSET
        else:
            completed_at = datetime.datetime.fromisoformat(_completed_at)




        deferral_window = d.pop("deferralWindow", UNSET)

        duration_seconds = d.pop("durationSeconds", UNSET)

        eta_seconds = d.pop("etaSeconds", UNSET)

        _finishes_at = d.pop("finishesAt", UNSET)
        finishes_at: datetime.datetime | Unset
        if isinstance(_finishes_at,  Unset):
            finishes_at = UNSET
        else:
            finishes_at = datetime.datetime.fromisoformat(_finishes_at)




        _helpers = d.pop("helpers", UNSET)
        helpers: LiveHelpers | Unset
        if isinstance(_helpers,  Unset):
            helpers = UNSET
        else:
            helpers = LiveHelpers.from_dict(_helpers)




        _in_doubt = d.pop("inDoubt", UNSET)
        in_doubt: LiveInDoubt | Unset
        if isinstance(_in_doubt,  Unset):
            in_doubt = UNSET
        else:
            in_doubt = LiveInDoubt.from_dict(_in_doubt)




        _incident = d.pop("incident", UNSET)
        incident: LiveIncident | Unset
        if isinstance(_incident,  Unset):
            incident = UNSET
        else:
            incident = LiveIncident.from_dict(_incident)




        _last_event_at = d.pop("lastEventAt", UNSET)
        last_event_at: datetime.datetime | Unset
        if isinstance(_last_event_at,  Unset):
            last_event_at = UNSET
        else:
            last_event_at = datetime.datetime.fromisoformat(_last_event_at)




        _paused_at = d.pop("pausedAt", UNSET)
        paused_at: datetime.datetime | Unset
        if isinstance(_paused_at,  Unset):
            paused_at = UNSET
        else:
            paused_at = datetime.datetime.fromisoformat(_paused_at)




        _postmaster = d.pop("postmaster", UNSET)
        postmaster: LivePostmaster | Unset
        if isinstance(_postmaster,  Unset):
            postmaster = UNSET
        else:
            postmaster = LivePostmaster.from_dict(_postmaster)




        _prepared_at = d.pop("preparedAt", UNSET)
        prepared_at: datetime.datetime | Unset
        if isinstance(_prepared_at,  Unset):
            prepared_at = UNSET
        else:
            prepared_at = datetime.datetime.fromisoformat(_prepared_at)




        _scheduled_at = d.pop("scheduledAt", UNSET)
        scheduled_at: datetime.datetime | Unset
        if isinstance(_scheduled_at,  Unset):
            scheduled_at = UNSET
        else:
            scheduled_at = datetime.datetime.fromisoformat(_scheduled_at)




        _started_at = d.pop("startedAt", UNSET)
        started_at: datetime.datetime | Unset
        if isinstance(_started_at,  Unset):
            started_at = UNSET
        else:
            started_at = datetime.datetime.fromisoformat(_started_at)




        _watch = d.pop("watch", UNSET)
        watch: LiveWatch | Unset
        if isinstance(_watch,  Unset):
            watch = UNSET
        else:
            watch = LiveWatch.from_dict(_watch)




        live_snapshot = cls(
            approved_by=approved_by,
            at=at,
            campaign_id=campaign_id,
            continuing=continuing,
            cost=cost,
            delivered_mix=delivered_mix,
            details=details,
            error=error,
            events=events,
            events_via=events_via,
            excluded=excluded,
            failovers=failovers,
            held=held,
            incidents=incidents,
            limits=limits,
            mailboxes=mailboxes,
            mailboxes_held=mailboxes_held,
            name=name,
            options=options,
            pause_reason=pause_reason,
            pause_scope_hint=pause_scope_hint,
            paused_by=paused_by,
            percent=percent,
            providers=providers,
            providers_held=providers_held,
            queued=queued,
            rate=rate,
            recipients=recipients,
            resumed=resumed,
            retrying=retrying,
            routes=routes,
            scheduled_by=scheduled_by,
            sent=sent,
            shared_streams=shared_streams,
            started_by=started_by,
            state=state,
            status=status,
            threshold_scope=threshold_scope,
            thresholds=thresholds,
            throttles=throttles,
            totals=totals,
            unsent=unsent,
            version=version,
            waiting=waiting,
            ab=ab,
            approved_at=approved_at,
            canary=canary,
            completed_at=completed_at,
            deferral_window=deferral_window,
            duration_seconds=duration_seconds,
            eta_seconds=eta_seconds,
            finishes_at=finishes_at,
            helpers=helpers,
            in_doubt=in_doubt,
            incident=incident,
            last_event_at=last_event_at,
            paused_at=paused_at,
            postmaster=postmaster,
            prepared_at=prepared_at,
            scheduled_at=scheduled_at,
            started_at=started_at,
            watch=watch,
        )

        return live_snapshot

