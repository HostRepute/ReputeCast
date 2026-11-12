from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_composer_offer_period import EarnComposerOfferPeriod
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="EarnComposerOffer")



@_attrs_define
class EarnComposerOffer:
    """ 
        Attributes:
            button (str):
            code (str):
            line (str):
            name (str):
            option (str):
            period (EarnComposerOfferPeriod):
            ends_at (datetime.datetime | Unset):
     """

    button: str
    code: str
    line: str
    name: str
    option: str
    period: EarnComposerOfferPeriod
    ends_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        button = self.button

        code = self.code

        line = self.line

        name = self.name

        option = self.option

        period = self.period.value

        ends_at: str | Unset = UNSET
        if not isinstance(self.ends_at, Unset):
            ends_at = self.ends_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "button": button,
            "code": code,
            "line": line,
            "name": name,
            "option": option,
            "period": period,
        })
        if ends_at is not UNSET:
            field_dict["endsAt"] = ends_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        button = d.pop("button")

        code = d.pop("code")

        line = d.pop("line")

        name = d.pop("name")

        option = d.pop("option")

        period = EarnComposerOfferPeriod(d.pop("period"))




        _ends_at = d.pop("endsAt", UNSET)
        ends_at: datetime.datetime | Unset
        if isinstance(_ends_at,  Unset):
            ends_at = UNSET
        else:
            ends_at = datetime.datetime.fromisoformat(_ends_at)




        earn_composer_offer = cls(
            button=button,
            code=code,
            line=line,
            name=name,
            option=option,
            period=period,
            ends_at=ends_at,
        )

        return earn_composer_offer

