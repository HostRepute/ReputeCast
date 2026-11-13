from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_team_seat_status import EarnTeamSeatStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="EarnTeamSeat")



@_attrs_define
class EarnTeamSeat:
    """ 
        Attributes:
            email (str):
            id (str):
            invited_at (datetime.datetime):
            name (str):
            status (EarnTeamSeatStatus):
            accepted_at (datetime.datetime | Unset):
            invite_expires_at (datetime.datetime | Unset):
            member_id (str | Unset):
     """

    email: str
    id: str
    invited_at: datetime.datetime
    name: str
    status: EarnTeamSeatStatus
    accepted_at: datetime.datetime | Unset = UNSET
    invite_expires_at: datetime.datetime | Unset = UNSET
    member_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        email = self.email

        id = self.id

        invited_at = self.invited_at.isoformat()

        name = self.name

        status = self.status.value

        accepted_at: str | Unset = UNSET
        if not isinstance(self.accepted_at, Unset):
            accepted_at = self.accepted_at.isoformat()

        invite_expires_at: str | Unset = UNSET
        if not isinstance(self.invite_expires_at, Unset):
            invite_expires_at = self.invite_expires_at.isoformat()

        member_id = self.member_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "email": email,
            "id": id,
            "invitedAt": invited_at,
            "name": name,
            "status": status,
        })
        if accepted_at is not UNSET:
            field_dict["acceptedAt"] = accepted_at
        if invite_expires_at is not UNSET:
            field_dict["inviteExpiresAt"] = invite_expires_at
        if member_id is not UNSET:
            field_dict["memberId"] = member_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        email = d.pop("email")

        id = d.pop("id")

        invited_at = datetime.datetime.fromisoformat(d.pop("invitedAt"))




        name = d.pop("name")

        status = EarnTeamSeatStatus(d.pop("status"))




        _accepted_at = d.pop("acceptedAt", UNSET)
        accepted_at: datetime.datetime | Unset
        if isinstance(_accepted_at,  Unset):
            accepted_at = UNSET
        else:
            accepted_at = datetime.datetime.fromisoformat(_accepted_at)




        _invite_expires_at = d.pop("inviteExpiresAt", UNSET)
        invite_expires_at: datetime.datetime | Unset
        if isinstance(_invite_expires_at,  Unset):
            invite_expires_at = UNSET
        else:
            invite_expires_at = datetime.datetime.fromisoformat(_invite_expires_at)




        member_id = d.pop("memberId", UNSET)

        earn_team_seat = cls(
            email=email,
            id=id,
            invited_at=invited_at,
            name=name,
            status=status,
            accepted_at=accepted_at,
            invite_expires_at=invite_expires_at,
            member_id=member_id,
        )

        return earn_team_seat

