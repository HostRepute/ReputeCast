from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.live_incident_kind import LiveIncidentKind
from ..models.live_incident_scope import LiveIncidentScope
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.incident_detail import IncidentDetail
  from ..models.incident_rate import IncidentRate





T = TypeVar("T", bound="LiveIncident")



@_attrs_define
class LiveIncident:
    """ 
        Attributes:
            detail (IncidentDetail):
            held (int):
            id (str):
            kind (LiveIncidentKind):
            mailbox (str):
            mailbox_name (str):
            metric (str):
            provider (str):
            provider_id (str):
            provider_short (str):
            resolution (str):
            scope (LiveIncidentScope):
            tripped_at (datetime.datetime):
            limit (float | Unset):
            resolved_at (datetime.datetime | Unset):
            since (IncidentRate | Unset):
            value (float | Unset):
     """

    detail: IncidentDetail
    held: int
    id: str
    kind: LiveIncidentKind
    mailbox: str
    mailbox_name: str
    metric: str
    provider: str
    provider_id: str
    provider_short: str
    resolution: str
    scope: LiveIncidentScope
    tripped_at: datetime.datetime
    limit: float | Unset = UNSET
    resolved_at: datetime.datetime | Unset = UNSET
    since: IncidentRate | Unset = UNSET
    value: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.incident_detail import IncidentDetail # noqa: PLC0415
        from ..models.incident_rate import IncidentRate # noqa: PLC0415
        detail = self.detail.to_dict()

        held = self.held

        id = self.id

        kind = self.kind.value

        mailbox = self.mailbox

        mailbox_name = self.mailbox_name

        metric = self.metric

        provider = self.provider

        provider_id = self.provider_id

        provider_short = self.provider_short

        resolution = self.resolution

        scope = self.scope.value

        tripped_at = self.tripped_at.isoformat()

        limit = self.limit

        resolved_at: str | Unset = UNSET
        if not isinstance(self.resolved_at, Unset):
            resolved_at = self.resolved_at.isoformat()

        since: dict[str, Any] | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.to_dict()

        value = self.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "detail": detail,
            "held": held,
            "id": id,
            "kind": kind,
            "mailbox": mailbox,
            "mailboxName": mailbox_name,
            "metric": metric,
            "provider": provider,
            "providerId": provider_id,
            "providerShort": provider_short,
            "resolution": resolution,
            "scope": scope,
            "trippedAt": tripped_at,
        })
        if limit is not UNSET:
            field_dict["limit"] = limit
        if resolved_at is not UNSET:
            field_dict["resolvedAt"] = resolved_at
        if since is not UNSET:
            field_dict["since"] = since
        if value is not UNSET:
            field_dict["value"] = value

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.incident_detail import IncidentDetail # noqa: PLC0415
        from ..models.incident_rate import IncidentRate # noqa: PLC0415
        d = dict(src_dict)
        detail = IncidentDetail.from_dict(d.pop("detail"))




        held = d.pop("held")

        id = d.pop("id")

        kind = LiveIncidentKind(d.pop("kind"))




        mailbox = d.pop("mailbox")

        mailbox_name = d.pop("mailboxName")

        metric = d.pop("metric")

        provider = d.pop("provider")

        provider_id = d.pop("providerId")

        provider_short = d.pop("providerShort")

        resolution = d.pop("resolution")

        scope = LiveIncidentScope(d.pop("scope"))




        tripped_at = datetime.datetime.fromisoformat(d.pop("trippedAt"))




        limit = d.pop("limit", UNSET)

        _resolved_at = d.pop("resolvedAt", UNSET)
        resolved_at: datetime.datetime | Unset
        if isinstance(_resolved_at,  Unset):
            resolved_at = UNSET
        else:
            resolved_at = datetime.datetime.fromisoformat(_resolved_at)




        _since = d.pop("since", UNSET)
        since: IncidentRate | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = IncidentRate.from_dict(_since)




        value = d.pop("value", UNSET)

        live_incident = cls(
            detail=detail,
            held=held,
            id=id,
            kind=kind,
            mailbox=mailbox,
            mailbox_name=mailbox_name,
            metric=metric,
            provider=provider,
            provider_id=provider_id,
            provider_short=provider_short,
            resolution=resolution,
            scope=scope,
            tripped_at=tripped_at,
            limit=limit,
            resolved_at=resolved_at,
            since=since,
            value=value,
        )

        return live_incident

