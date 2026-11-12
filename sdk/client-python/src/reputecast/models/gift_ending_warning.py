from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="GiftEndingWarning")



@_attrs_define
class GiftEndingWarning:
    """ 
        Attributes:
            buyer (str):
            due (datetime.datetime):
            in_person (bool):
            recipient (str):
            template_id (str | Unset):
     """

    buyer: str
    due: datetime.datetime
    in_person: bool
    recipient: str
    template_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        buyer = self.buyer

        due = self.due.isoformat()

        in_person = self.in_person

        recipient = self.recipient

        template_id = self.template_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "buyer": buyer,
            "due": due,
            "inPerson": in_person,
            "recipient": recipient,
        })
        if template_id is not UNSET:
            field_dict["templateId"] = template_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        buyer = d.pop("buyer")

        due = datetime.datetime.fromisoformat(d.pop("due"))




        in_person = d.pop("inPerson")

        recipient = d.pop("recipient")

        template_id = d.pop("templateId", UNSET)

        gift_ending_warning = cls(
            buyer=buyer,
            due=due,
            in_person=in_person,
            recipient=recipient,
            template_id=template_id,
        )

        return gift_ending_warning

