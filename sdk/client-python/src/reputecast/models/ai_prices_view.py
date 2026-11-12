from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="AIPricesView")



@_attrs_define
class AIPricesView:
    """ 
        Attributes:
            input_ (float): Dollars per million input tokens
            output (float): Dollars per million output tokens
            from_ (datetime.datetime | Unset):
     """

    input_: float
    output: float
    from_: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        input_ = self.input_

        output = self.output

        from_: str | Unset = UNSET
        if not isinstance(self.from_, Unset):
            from_ = self.from_.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "input": input_,
            "output": output,
        })
        if from_ is not UNSET:
            field_dict["from"] = from_

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        input_ = d.pop("input")

        output = d.pop("output")

        _from_ = d.pop("from", UNSET)
        from_: datetime.datetime | Unset
        if isinstance(_from_,  Unset):
            from_ = UNSET
        else:
            from_ = datetime.datetime.fromisoformat(_from_)




        ai_prices_view = cls(
            input_=input_,
            output=output,
            from_=from_,
        )

        return ai_prices_view

