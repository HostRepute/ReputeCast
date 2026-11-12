from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.verification_job_view_kind import VerificationJobViewKind
from ..models.verification_job_view_status import VerificationJobViewStatus
from ..models.verification_job_view_stop_reason import VerificationJobViewStopReason
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.verification_job_view_reasons import VerificationJobViewReasons





T = TypeVar("T", bound="VerificationJobView")



@_attrs_define
class VerificationJobView:
    """ 
        Attributes:
            can_export (bool):
            can_verify (bool):
            checked (int):
            cost (float):
            created_at (datetime.datetime):
            estimated (float):
            id (str):
            invalid (int):
            kind (VerificationJobViewKind):
            label (str):
            origin (str):
            price (float):
            reasons (VerificationJobViewReasons):
            region_label (str):
            risky (int):
            service (str):
            service_name (str):
            skipped (int):
            started_by (str):
            status (VerificationJobViewStatus):
            stop_reason (VerificationJobViewStopReason):
            suppressed (int):
            tiers (list[str]):
            total (int):
            unknown (int):
            valid (int):
            balance_after (int | Unset):
            balance_before (int | Unset):
            campaign_id (str | Unset):
            error_since (datetime.datetime | Unset):
            eta_seconds (int | Unset):
            finished_at (datetime.datetime | Unset):
            import_id (str | Unset):
            last_error (str | Unset):
            list_id (str | Unset):
            next_try_at (datetime.datetime | Unset):
            recheck_from (datetime.datetime | Unset):
            retry_until (datetime.datetime | Unset):
            started_at (datetime.datetime | Unset):
            stopped_by (str | Unset):
     """

    can_export: bool
    can_verify: bool
    checked: int
    cost: float
    created_at: datetime.datetime
    estimated: float
    id: str
    invalid: int
    kind: VerificationJobViewKind
    label: str
    origin: str
    price: float
    reasons: VerificationJobViewReasons
    region_label: str
    risky: int
    service: str
    service_name: str
    skipped: int
    started_by: str
    status: VerificationJobViewStatus
    stop_reason: VerificationJobViewStopReason
    suppressed: int
    tiers: list[str]
    total: int
    unknown: int
    valid: int
    balance_after: int | Unset = UNSET
    balance_before: int | Unset = UNSET
    campaign_id: str | Unset = UNSET
    error_since: datetime.datetime | Unset = UNSET
    eta_seconds: int | Unset = UNSET
    finished_at: datetime.datetime | Unset = UNSET
    import_id: str | Unset = UNSET
    last_error: str | Unset = UNSET
    list_id: str | Unset = UNSET
    next_try_at: datetime.datetime | Unset = UNSET
    recheck_from: datetime.datetime | Unset = UNSET
    retry_until: datetime.datetime | Unset = UNSET
    started_at: datetime.datetime | Unset = UNSET
    stopped_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.verification_job_view_reasons import VerificationJobViewReasons # noqa: PLC0415
        can_export = self.can_export

        can_verify = self.can_verify

        checked = self.checked

        cost = self.cost

        created_at = self.created_at.isoformat()

        estimated = self.estimated

        id = self.id

        invalid = self.invalid

        kind = self.kind.value

        label = self.label

        origin = self.origin

        price = self.price

        reasons = self.reasons.to_dict()

        region_label = self.region_label

        risky = self.risky

        service = self.service

        service_name = self.service_name

        skipped = self.skipped

        started_by = self.started_by

        status = self.status.value

        stop_reason = self.stop_reason.value

        suppressed = self.suppressed

        tiers = self.tiers



        total = self.total

        unknown = self.unknown

        valid = self.valid

        balance_after = self.balance_after

        balance_before = self.balance_before

        campaign_id = self.campaign_id

        error_since: str | Unset = UNSET
        if not isinstance(self.error_since, Unset):
            error_since = self.error_since.isoformat()

        eta_seconds = self.eta_seconds

        finished_at: str | Unset = UNSET
        if not isinstance(self.finished_at, Unset):
            finished_at = self.finished_at.isoformat()

        import_id = self.import_id

        last_error = self.last_error

        list_id = self.list_id

        next_try_at: str | Unset = UNSET
        if not isinstance(self.next_try_at, Unset):
            next_try_at = self.next_try_at.isoformat()

        recheck_from: str | Unset = UNSET
        if not isinstance(self.recheck_from, Unset):
            recheck_from = self.recheck_from.isoformat()

        retry_until: str | Unset = UNSET
        if not isinstance(self.retry_until, Unset):
            retry_until = self.retry_until.isoformat()

        started_at: str | Unset = UNSET
        if not isinstance(self.started_at, Unset):
            started_at = self.started_at.isoformat()

        stopped_by = self.stopped_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canExport": can_export,
            "canVerify": can_verify,
            "checked": checked,
            "cost": cost,
            "createdAt": created_at,
            "estimated": estimated,
            "id": id,
            "invalid": invalid,
            "kind": kind,
            "label": label,
            "origin": origin,
            "price": price,
            "reasons": reasons,
            "regionLabel": region_label,
            "risky": risky,
            "service": service,
            "serviceName": service_name,
            "skipped": skipped,
            "startedBy": started_by,
            "status": status,
            "stopReason": stop_reason,
            "suppressed": suppressed,
            "tiers": tiers,
            "total": total,
            "unknown": unknown,
            "valid": valid,
        })
        if balance_after is not UNSET:
            field_dict["balanceAfter"] = balance_after
        if balance_before is not UNSET:
            field_dict["balanceBefore"] = balance_before
        if campaign_id is not UNSET:
            field_dict["campaignId"] = campaign_id
        if error_since is not UNSET:
            field_dict["errorSince"] = error_since
        if eta_seconds is not UNSET:
            field_dict["etaSeconds"] = eta_seconds
        if finished_at is not UNSET:
            field_dict["finishedAt"] = finished_at
        if import_id is not UNSET:
            field_dict["importId"] = import_id
        if last_error is not UNSET:
            field_dict["lastError"] = last_error
        if list_id is not UNSET:
            field_dict["listId"] = list_id
        if next_try_at is not UNSET:
            field_dict["nextTryAt"] = next_try_at
        if recheck_from is not UNSET:
            field_dict["recheckFrom"] = recheck_from
        if retry_until is not UNSET:
            field_dict["retryUntil"] = retry_until
        if started_at is not UNSET:
            field_dict["startedAt"] = started_at
        if stopped_by is not UNSET:
            field_dict["stoppedBy"] = stopped_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.verification_job_view_reasons import VerificationJobViewReasons # noqa: PLC0415
        d = dict(src_dict)
        can_export = d.pop("canExport")

        can_verify = d.pop("canVerify")

        checked = d.pop("checked")

        cost = d.pop("cost")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        estimated = d.pop("estimated")

        id = d.pop("id")

        invalid = d.pop("invalid")

        kind = VerificationJobViewKind(d.pop("kind"))




        label = d.pop("label")

        origin = d.pop("origin")

        price = d.pop("price")

        reasons = VerificationJobViewReasons.from_dict(d.pop("reasons"))




        region_label = d.pop("regionLabel")

        risky = d.pop("risky")

        service = d.pop("service")

        service_name = d.pop("serviceName")

        skipped = d.pop("skipped")

        started_by = d.pop("startedBy")

        status = VerificationJobViewStatus(d.pop("status"))




        stop_reason = VerificationJobViewStopReason(d.pop("stopReason"))




        suppressed = d.pop("suppressed")

        tiers = cast(list[str], d.pop("tiers"))


        total = d.pop("total")

        unknown = d.pop("unknown")

        valid = d.pop("valid")

        balance_after = d.pop("balanceAfter", UNSET)

        balance_before = d.pop("balanceBefore", UNSET)

        campaign_id = d.pop("campaignId", UNSET)

        _error_since = d.pop("errorSince", UNSET)
        error_since: datetime.datetime | Unset
        if isinstance(_error_since,  Unset):
            error_since = UNSET
        else:
            error_since = datetime.datetime.fromisoformat(_error_since)




        eta_seconds = d.pop("etaSeconds", UNSET)

        _finished_at = d.pop("finishedAt", UNSET)
        finished_at: datetime.datetime | Unset
        if isinstance(_finished_at,  Unset):
            finished_at = UNSET
        else:
            finished_at = datetime.datetime.fromisoformat(_finished_at)




        import_id = d.pop("importId", UNSET)

        last_error = d.pop("lastError", UNSET)

        list_id = d.pop("listId", UNSET)

        _next_try_at = d.pop("nextTryAt", UNSET)
        next_try_at: datetime.datetime | Unset
        if isinstance(_next_try_at,  Unset):
            next_try_at = UNSET
        else:
            next_try_at = datetime.datetime.fromisoformat(_next_try_at)




        _recheck_from = d.pop("recheckFrom", UNSET)
        recheck_from: datetime.datetime | Unset
        if isinstance(_recheck_from,  Unset):
            recheck_from = UNSET
        else:
            recheck_from = datetime.datetime.fromisoformat(_recheck_from)




        _retry_until = d.pop("retryUntil", UNSET)
        retry_until: datetime.datetime | Unset
        if isinstance(_retry_until,  Unset):
            retry_until = UNSET
        else:
            retry_until = datetime.datetime.fromisoformat(_retry_until)




        _started_at = d.pop("startedAt", UNSET)
        started_at: datetime.datetime | Unset
        if isinstance(_started_at,  Unset):
            started_at = UNSET
        else:
            started_at = datetime.datetime.fromisoformat(_started_at)




        stopped_by = d.pop("stoppedBy", UNSET)

        verification_job_view = cls(
            can_export=can_export,
            can_verify=can_verify,
            checked=checked,
            cost=cost,
            created_at=created_at,
            estimated=estimated,
            id=id,
            invalid=invalid,
            kind=kind,
            label=label,
            origin=origin,
            price=price,
            reasons=reasons,
            region_label=region_label,
            risky=risky,
            service=service,
            service_name=service_name,
            skipped=skipped,
            started_by=started_by,
            status=status,
            stop_reason=stop_reason,
            suppressed=suppressed,
            tiers=tiers,
            total=total,
            unknown=unknown,
            valid=valid,
            balance_after=balance_after,
            balance_before=balance_before,
            campaign_id=campaign_id,
            error_since=error_since,
            eta_seconds=eta_seconds,
            finished_at=finished_at,
            import_id=import_id,
            last_error=last_error,
            list_id=list_id,
            next_try_at=next_try_at,
            recheck_from=recheck_from,
            retry_until=retry_until,
            started_at=started_at,
            stopped_by=stopped_by,
        )

        return verification_job_view

