from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_member_team_role import EarnMemberTeamRole
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="EarnMemberTeam")



@_attrs_define
class EarnMemberTeam:
    """ 
        Attributes:
            id (str):
            invites (int):
            name (str):
            role (EarnMemberTeamRole):
            seats (int):
            used (int):
            buyer_id (str | Unset):
            buyer_name (str | Unset):
            ended_at (datetime.datetime | Unset):
            renews_at (datetime.datetime | Unset):
            seat_id (str | Unset):
     """

    id: str
    invites: int
    name: str
    role: EarnMemberTeamRole
    seats: int
    used: int
    buyer_id: str | Unset = UNSET
    buyer_name: str | Unset = UNSET
    ended_at: datetime.datetime | Unset = UNSET
    renews_at: datetime.datetime | Unset = UNSET
    seat_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        invites = self.invites

        name = self.name

        role = self.role.value

        seats = self.seats

        used = self.used

        buyer_id = self.buyer_id

        buyer_name = self.buyer_name

        ended_at: str | Unset = UNSET
        if not isinstance(self.ended_at, Unset):
            ended_at = self.ended_at.isoformat()

        renews_at: str | Unset = UNSET
        if not isinstance(self.renews_at, Unset):
            renews_at = self.renews_at.isoformat()

        seat_id = self.seat_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "invites": invites,
            "name": name,
            "role": role,
            "seats": seats,
            "used": used,
        })
        if buyer_id is not UNSET:
            field_dict["buyerId"] = buyer_id
        if buyer_name is not UNSET:
            field_dict["buyerName"] = buyer_name
        if ended_at is not UNSET:
            field_dict["endedAt"] = ended_at
        if renews_at is not UNSET:
            field_dict["renewsAt"] = renews_at
        if seat_id is not UNSET:
            field_dict["seatId"] = seat_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        invites = d.pop("invites")

        name = d.pop("name")

        role = EarnMemberTeamRole(d.pop("role"))




        seats = d.pop("seats")

        used = d.pop("used")

        buyer_id = d.pop("buyerId", UNSET)

        buyer_name = d.pop("buyerName", UNSET)

        _ended_at = d.pop("endedAt", UNSET)
        ended_at: datetime.datetime | Unset
        if isinstance(_ended_at,  Unset):
            ended_at = UNSET
        else:
            ended_at = datetime.datetime.fromisoformat(_ended_at)




        _renews_at = d.pop("renewsAt", UNSET)
        renews_at: datetime.datetime | Unset
        if isinstance(_renews_at,  Unset):
            renews_at = UNSET
        else:
            renews_at = datetime.datetime.fromisoformat(_renews_at)




        seat_id = d.pop("seatId", UNSET)

        earn_member_team = cls(
            id=id,
            invites=invites,
            name=name,
            role=role,
            seats=seats,
            used=used,
            buyer_id=buyer_id,
            buyer_name=buyer_name,
            ended_at=ended_at,
            renews_at=renews_at,
            seat_id=seat_id,
        )

        return earn_member_team

