from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.shell_host_repute_status import ShellHostReputeStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ShellHostRepute")



@_attrs_define
class ShellHostRepute:
    """ 
        Attributes:
            status (ShellHostReputeStatus):
            team_name (str):
            last_sync_at (datetime.datetime | Unset):
     """

    status: ShellHostReputeStatus
    team_name: str
    last_sync_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        status = self.status.value

        team_name = self.team_name

        last_sync_at: str | Unset = UNSET
        if not isinstance(self.last_sync_at, Unset):
            last_sync_at = self.last_sync_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "status": status,
            "teamName": team_name,
        })
        if last_sync_at is not UNSET:
            field_dict["lastSyncAt"] = last_sync_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        status = ShellHostReputeStatus(d.pop("status"))




        team_name = d.pop("teamName")

        _last_sync_at = d.pop("lastSyncAt", UNSET)
        last_sync_at: datetime.datetime | Unset
        if isinstance(_last_sync_at,  Unset):
            last_sync_at = UNSET
        else:
            last_sync_at = datetime.datetime.fromisoformat(_last_sync_at)




        shell_host_repute = cls(
            status=status,
            team_name=team_name,
            last_sync_at=last_sync_at,
        )

        return shell_host_repute

