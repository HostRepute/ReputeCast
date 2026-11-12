from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.deliverability_host_repute_status import DeliverabilityHostReputeStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.host_repute_outage import HostReputeOutage





T = TypeVar("T", bound="DeliverabilityHostRepute")



@_attrs_define
class DeliverabilityHostRepute:
    """ 
        Attributes:
            interval_minutes (int):
            sources (int):
            status (DeliverabilityHostReputeStatus):
            dmarc_rua (str | Unset):
            last_sync_at (datetime.datetime | Unset):
            team_name (str | Unset):
            token_rejected_at (datetime.datetime | Unset):
            unreachable (HostReputeOutage | Unset):
     """

    interval_minutes: int
    sources: int
    status: DeliverabilityHostReputeStatus
    dmarc_rua: str | Unset = UNSET
    last_sync_at: datetime.datetime | Unset = UNSET
    team_name: str | Unset = UNSET
    token_rejected_at: datetime.datetime | Unset = UNSET
    unreachable: HostReputeOutage | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.host_repute_outage import HostReputeOutage # noqa: PLC0415
        interval_minutes = self.interval_minutes

        sources = self.sources

        status = self.status.value

        dmarc_rua = self.dmarc_rua

        last_sync_at: str | Unset = UNSET
        if not isinstance(self.last_sync_at, Unset):
            last_sync_at = self.last_sync_at.isoformat()

        team_name = self.team_name

        token_rejected_at: str | Unset = UNSET
        if not isinstance(self.token_rejected_at, Unset):
            token_rejected_at = self.token_rejected_at.isoformat()

        unreachable: dict[str, Any] | Unset = UNSET
        if not isinstance(self.unreachable, Unset):
            unreachable = self.unreachable.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "intervalMinutes": interval_minutes,
            "sources": sources,
            "status": status,
        })
        if dmarc_rua is not UNSET:
            field_dict["dmarcRua"] = dmarc_rua
        if last_sync_at is not UNSET:
            field_dict["lastSyncAt"] = last_sync_at
        if team_name is not UNSET:
            field_dict["teamName"] = team_name
        if token_rejected_at is not UNSET:
            field_dict["tokenRejectedAt"] = token_rejected_at
        if unreachable is not UNSET:
            field_dict["unreachable"] = unreachable

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.host_repute_outage import HostReputeOutage # noqa: PLC0415
        d = dict(src_dict)
        interval_minutes = d.pop("intervalMinutes")

        sources = d.pop("sources")

        status = DeliverabilityHostReputeStatus(d.pop("status"))




        dmarc_rua = d.pop("dmarcRua", UNSET)

        _last_sync_at = d.pop("lastSyncAt", UNSET)
        last_sync_at: datetime.datetime | Unset
        if isinstance(_last_sync_at,  Unset):
            last_sync_at = UNSET
        else:
            last_sync_at = datetime.datetime.fromisoformat(_last_sync_at)




        team_name = d.pop("teamName", UNSET)

        _token_rejected_at = d.pop("tokenRejectedAt", UNSET)
        token_rejected_at: datetime.datetime | Unset
        if isinstance(_token_rejected_at,  Unset):
            token_rejected_at = UNSET
        else:
            token_rejected_at = datetime.datetime.fromisoformat(_token_rejected_at)




        _unreachable = d.pop("unreachable", UNSET)
        unreachable: HostReputeOutage | Unset
        if isinstance(_unreachable,  Unset):
            unreachable = UNSET
        else:
            unreachable = HostReputeOutage.from_dict(_unreachable)




        deliverability_host_repute = cls(
            interval_minutes=interval_minutes,
            sources=sources,
            status=status,
            dmarc_rua=dmarc_rua,
            last_sync_at=last_sync_at,
            team_name=team_name,
            token_rejected_at=token_rejected_at,
            unreachable=unreachable,
        )

        return deliverability_host_repute

