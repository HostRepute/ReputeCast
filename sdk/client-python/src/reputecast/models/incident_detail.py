from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.incident_detail_action import IncidentDetailAction
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.in_doubt_share import InDoubtShare
  from ..models.incident_cause import IncidentCause
  from ..models.incident_log_line import IncidentLogLine
  from ..models.incident_option import IncidentOption
  from ..models.incident_rate import IncidentRate
  from ..models.move_estimate import MoveEstimate
  from ..models.resume_changes import ResumeChanges





T = TypeVar("T", bound="IncidentDetail")



@_attrs_define
class IncidentDetail:
    """ 
        Attributes:
            succeeded (int):
            action (IncidentDetailAction | Unset):
            attempts (int | Unset):
            base (int | Unset):
            campaign (IncidentRate | Unset):
            causes (list[IncidentCause] | Unset):
            changes (ResumeChanges | Unset):
            clear (bool | Unset):
            continuing (list[str] | Unset):
            delivered (int | Unset):
            error_kind (str | Unset):
            error_rate (float | Unset):
            events (int | Unset):
            from_ (datetime.datetime | Unset):
            host (str | Unset):
            in_doubt (list[InDoubtShare] | Unset):
            log (list[IncidentLogLine] | Unset):
            move (MoveEstimate | Unset):
            note (str | Unset):
            options (list[IncidentOption] | Unset):
            policy (str | Unset):
            probed (bool | Unset):
            reason (str | Unset):
            recommendations (list[str] | Unset):
            remaining (int | Unset):
            retried (bool | Unset):
            secondary (IncidentRate | Unset):
            sent (int | Unset):
            sent_on_route (int | Unset):
            slowed_to (float | Unset):
            throttle (float | Unset):
            to (datetime.datetime | Unset):
            total (int | Unset):
            warmup_note (str | Unset):
            window (int | Unset):
     """

    succeeded: int
    action: IncidentDetailAction | Unset = UNSET
    attempts: int | Unset = UNSET
    base: int | Unset = UNSET
    campaign: IncidentRate | Unset = UNSET
    causes: list[IncidentCause] | Unset = UNSET
    changes: ResumeChanges | Unset = UNSET
    clear: bool | Unset = UNSET
    continuing: list[str] | Unset = UNSET
    delivered: int | Unset = UNSET
    error_kind: str | Unset = UNSET
    error_rate: float | Unset = UNSET
    events: int | Unset = UNSET
    from_: datetime.datetime | Unset = UNSET
    host: str | Unset = UNSET
    in_doubt: list[InDoubtShare] | Unset = UNSET
    log: list[IncidentLogLine] | Unset = UNSET
    move: MoveEstimate | Unset = UNSET
    note: str | Unset = UNSET
    options: list[IncidentOption] | Unset = UNSET
    policy: str | Unset = UNSET
    probed: bool | Unset = UNSET
    reason: str | Unset = UNSET
    recommendations: list[str] | Unset = UNSET
    remaining: int | Unset = UNSET
    retried: bool | Unset = UNSET
    secondary: IncidentRate | Unset = UNSET
    sent: int | Unset = UNSET
    sent_on_route: int | Unset = UNSET
    slowed_to: float | Unset = UNSET
    throttle: float | Unset = UNSET
    to: datetime.datetime | Unset = UNSET
    total: int | Unset = UNSET
    warmup_note: str | Unset = UNSET
    window: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.in_doubt_share import InDoubtShare # noqa: PLC0415
        from ..models.incident_cause import IncidentCause # noqa: PLC0415
        from ..models.incident_log_line import IncidentLogLine # noqa: PLC0415
        from ..models.incident_option import IncidentOption # noqa: PLC0415
        from ..models.incident_rate import IncidentRate # noqa: PLC0415
        from ..models.move_estimate import MoveEstimate # noqa: PLC0415
        from ..models.resume_changes import ResumeChanges # noqa: PLC0415
        succeeded = self.succeeded

        action: str | Unset = UNSET
        if not isinstance(self.action, Unset):
            action = self.action.value


        attempts = self.attempts

        base = self.base

        campaign: dict[str, Any] | Unset = UNSET
        if not isinstance(self.campaign, Unset):
            campaign = self.campaign.to_dict()

        causes: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.causes, Unset):
            causes = []
            for causes_item_data in self.causes:
                causes_item = causes_item_data.to_dict()
                causes.append(causes_item)



        changes: dict[str, Any] | Unset = UNSET
        if not isinstance(self.changes, Unset):
            changes = self.changes.to_dict()

        clear = self.clear

        continuing: list[str] | Unset = UNSET
        if not isinstance(self.continuing, Unset):
            continuing = self.continuing



        delivered = self.delivered

        error_kind = self.error_kind

        error_rate = self.error_rate

        events = self.events

        from_: str | Unset = UNSET
        if not isinstance(self.from_, Unset):
            from_ = self.from_.isoformat()

        host = self.host

        in_doubt: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.in_doubt, Unset):
            in_doubt = []
            for in_doubt_item_data in self.in_doubt:
                in_doubt_item = in_doubt_item_data.to_dict()
                in_doubt.append(in_doubt_item)



        log: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.log, Unset):
            log = []
            for log_item_data in self.log:
                log_item = log_item_data.to_dict()
                log.append(log_item)



        move: dict[str, Any] | Unset = UNSET
        if not isinstance(self.move, Unset):
            move = self.move.to_dict()

        note = self.note

        options: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.options, Unset):
            options = []
            for options_item_data in self.options:
                options_item = options_item_data.to_dict()
                options.append(options_item)



        policy = self.policy

        probed = self.probed

        reason = self.reason

        recommendations: list[str] | Unset = UNSET
        if not isinstance(self.recommendations, Unset):
            recommendations = self.recommendations



        remaining = self.remaining

        retried = self.retried

        secondary: dict[str, Any] | Unset = UNSET
        if not isinstance(self.secondary, Unset):
            secondary = self.secondary.to_dict()

        sent = self.sent

        sent_on_route = self.sent_on_route

        slowed_to = self.slowed_to

        throttle = self.throttle

        to: str | Unset = UNSET
        if not isinstance(self.to, Unset):
            to = self.to.isoformat()

        total = self.total

        warmup_note = self.warmup_note

        window = self.window


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "succeeded": succeeded,
        })
        if action is not UNSET:
            field_dict["action"] = action
        if attempts is not UNSET:
            field_dict["attempts"] = attempts
        if base is not UNSET:
            field_dict["base"] = base
        if campaign is not UNSET:
            field_dict["campaign"] = campaign
        if causes is not UNSET:
            field_dict["causes"] = causes
        if changes is not UNSET:
            field_dict["changes"] = changes
        if clear is not UNSET:
            field_dict["clear"] = clear
        if continuing is not UNSET:
            field_dict["continuing"] = continuing
        if delivered is not UNSET:
            field_dict["delivered"] = delivered
        if error_kind is not UNSET:
            field_dict["errorKind"] = error_kind
        if error_rate is not UNSET:
            field_dict["errorRate"] = error_rate
        if events is not UNSET:
            field_dict["events"] = events
        if from_ is not UNSET:
            field_dict["from"] = from_
        if host is not UNSET:
            field_dict["host"] = host
        if in_doubt is not UNSET:
            field_dict["inDoubt"] = in_doubt
        if log is not UNSET:
            field_dict["log"] = log
        if move is not UNSET:
            field_dict["move"] = move
        if note is not UNSET:
            field_dict["note"] = note
        if options is not UNSET:
            field_dict["options"] = options
        if policy is not UNSET:
            field_dict["policy"] = policy
        if probed is not UNSET:
            field_dict["probed"] = probed
        if reason is not UNSET:
            field_dict["reason"] = reason
        if recommendations is not UNSET:
            field_dict["recommendations"] = recommendations
        if remaining is not UNSET:
            field_dict["remaining"] = remaining
        if retried is not UNSET:
            field_dict["retried"] = retried
        if secondary is not UNSET:
            field_dict["secondary"] = secondary
        if sent is not UNSET:
            field_dict["sent"] = sent
        if sent_on_route is not UNSET:
            field_dict["sentOnRoute"] = sent_on_route
        if slowed_to is not UNSET:
            field_dict["slowedTo"] = slowed_to
        if throttle is not UNSET:
            field_dict["throttle"] = throttle
        if to is not UNSET:
            field_dict["to"] = to
        if total is not UNSET:
            field_dict["total"] = total
        if warmup_note is not UNSET:
            field_dict["warmupNote"] = warmup_note
        if window is not UNSET:
            field_dict["window"] = window

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.in_doubt_share import InDoubtShare # noqa: PLC0415
        from ..models.incident_cause import IncidentCause # noqa: PLC0415
        from ..models.incident_log_line import IncidentLogLine # noqa: PLC0415
        from ..models.incident_option import IncidentOption # noqa: PLC0415
        from ..models.incident_rate import IncidentRate # noqa: PLC0415
        from ..models.move_estimate import MoveEstimate # noqa: PLC0415
        from ..models.resume_changes import ResumeChanges # noqa: PLC0415
        d = dict(src_dict)
        succeeded = d.pop("succeeded")

        _action = d.pop("action", UNSET)
        action: IncidentDetailAction | Unset
        if isinstance(_action,  Unset):
            action = UNSET
        else:
            action = IncidentDetailAction(_action)




        attempts = d.pop("attempts", UNSET)

        base = d.pop("base", UNSET)

        _campaign = d.pop("campaign", UNSET)
        campaign: IncidentRate | Unset
        if isinstance(_campaign,  Unset):
            campaign = UNSET
        else:
            campaign = IncidentRate.from_dict(_campaign)




        _causes = d.pop("causes", UNSET)
        causes: list[IncidentCause] | Unset = UNSET
        if _causes is not UNSET:
            causes = []
            for causes_item_data in _causes:
                causes_item = IncidentCause.from_dict(causes_item_data)



                causes.append(causes_item)


        _changes = d.pop("changes", UNSET)
        changes: ResumeChanges | Unset
        if isinstance(_changes,  Unset):
            changes = UNSET
        else:
            changes = ResumeChanges.from_dict(_changes)




        clear = d.pop("clear", UNSET)

        continuing = cast(list[str], d.pop("continuing", UNSET))


        delivered = d.pop("delivered", UNSET)

        error_kind = d.pop("errorKind", UNSET)

        error_rate = d.pop("errorRate", UNSET)

        events = d.pop("events", UNSET)

        _from_ = d.pop("from", UNSET)
        from_: datetime.datetime | Unset
        if isinstance(_from_,  Unset):
            from_ = UNSET
        else:
            from_ = datetime.datetime.fromisoformat(_from_)




        host = d.pop("host", UNSET)

        _in_doubt = d.pop("inDoubt", UNSET)
        in_doubt: list[InDoubtShare] | Unset = UNSET
        if _in_doubt is not UNSET:
            in_doubt = []
            for in_doubt_item_data in _in_doubt:
                in_doubt_item = InDoubtShare.from_dict(in_doubt_item_data)



                in_doubt.append(in_doubt_item)


        _log = d.pop("log", UNSET)
        log: list[IncidentLogLine] | Unset = UNSET
        if _log is not UNSET:
            log = []
            for log_item_data in _log:
                log_item = IncidentLogLine.from_dict(log_item_data)



                log.append(log_item)


        _move = d.pop("move", UNSET)
        move: MoveEstimate | Unset
        if isinstance(_move,  Unset):
            move = UNSET
        else:
            move = MoveEstimate.from_dict(_move)




        note = d.pop("note", UNSET)

        _options = d.pop("options", UNSET)
        options: list[IncidentOption] | Unset = UNSET
        if _options is not UNSET:
            options = []
            for options_item_data in _options:
                options_item = IncidentOption.from_dict(options_item_data)



                options.append(options_item)


        policy = d.pop("policy", UNSET)

        probed = d.pop("probed", UNSET)

        reason = d.pop("reason", UNSET)

        recommendations = cast(list[str], d.pop("recommendations", UNSET))


        remaining = d.pop("remaining", UNSET)

        retried = d.pop("retried", UNSET)

        _secondary = d.pop("secondary", UNSET)
        secondary: IncidentRate | Unset
        if isinstance(_secondary,  Unset):
            secondary = UNSET
        else:
            secondary = IncidentRate.from_dict(_secondary)




        sent = d.pop("sent", UNSET)

        sent_on_route = d.pop("sentOnRoute", UNSET)

        slowed_to = d.pop("slowedTo", UNSET)

        throttle = d.pop("throttle", UNSET)

        _to = d.pop("to", UNSET)
        to: datetime.datetime | Unset
        if isinstance(_to,  Unset):
            to = UNSET
        else:
            to = datetime.datetime.fromisoformat(_to)




        total = d.pop("total", UNSET)

        warmup_note = d.pop("warmupNote", UNSET)

        window = d.pop("window", UNSET)

        incident_detail = cls(
            succeeded=succeeded,
            action=action,
            attempts=attempts,
            base=base,
            campaign=campaign,
            causes=causes,
            changes=changes,
            clear=clear,
            continuing=continuing,
            delivered=delivered,
            error_kind=error_kind,
            error_rate=error_rate,
            events=events,
            from_=from_,
            host=host,
            in_doubt=in_doubt,
            log=log,
            move=move,
            note=note,
            options=options,
            policy=policy,
            probed=probed,
            reason=reason,
            recommendations=recommendations,
            remaining=remaining,
            retried=retried,
            secondary=secondary,
            sent=sent,
            sent_on_route=sent_on_route,
            slowed_to=slowed_to,
            throttle=throttle,
            to=to,
            total=total,
            warmup_note=warmup_note,
            window=window,
        )

        return incident_detail

