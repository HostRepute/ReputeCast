from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.sponsor_ref import SponsorRef





T = TypeVar("T", bound="SponsorPropose")



@_attrs_define
class SponsorPropose:
    """ 
        Attributes:
            agent (str):
            at (datetime.datetime):
            id (str):
            note (str):
            price_cents (int):
            sponsor (SponsorRef | Unset):
     """

    agent: str
    at: datetime.datetime
    id: str
    note: str
    price_cents: int
    sponsor: SponsorRef | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sponsor_ref import SponsorRef # noqa: PLC0415
        agent = self.agent

        at = self.at.isoformat()

        id = self.id

        note = self.note

        price_cents = self.price_cents

        sponsor: dict[str, Any] | Unset = UNSET
        if not isinstance(self.sponsor, Unset):
            sponsor = self.sponsor.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "agent": agent,
            "at": at,
            "id": id,
            "note": note,
            "priceCents": price_cents,
        })
        if sponsor is not UNSET:
            field_dict["sponsor"] = sponsor

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sponsor_ref import SponsorRef # noqa: PLC0415
        d = dict(src_dict)
        agent = d.pop("agent")

        at = datetime.datetime.fromisoformat(d.pop("at"))




        id = d.pop("id")

        note = d.pop("note")

        price_cents = d.pop("priceCents")

        _sponsor = d.pop("sponsor", UNSET)
        sponsor: SponsorRef | Unset
        if isinstance(_sponsor,  Unset):
            sponsor = UNSET
        else:
            sponsor = SponsorRef.from_dict(_sponsor)




        sponsor_propose = cls(
            agent=agent,
            at=at,
            id=id,
            note=note,
            price_cents=price_cents,
            sponsor=sponsor,
        )

        return sponsor_propose

