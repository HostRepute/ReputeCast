from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.conversion_result_matched import ConversionResultMatched
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.conversion_credit import ConversionCredit





T = TypeVar("T", bound="ConversionResult")



@_attrs_define
class ConversionResult:
    """ 
        Attributes:
            id (str):
            matched (ConversionResultMatched):
            credited (ConversionCredit | Unset):
            test (bool | Unset):
     """

    id: str
    matched: ConversionResultMatched
    credited: ConversionCredit | Unset = UNSET
    test: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.conversion_credit import ConversionCredit # noqa: PLC0415
        id = self.id

        matched = self.matched.value

        credited: dict[str, Any] | Unset = UNSET
        if not isinstance(self.credited, Unset):
            credited = self.credited.to_dict()

        test = self.test


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "matched": matched,
        })
        if credited is not UNSET:
            field_dict["credited"] = credited
        if test is not UNSET:
            field_dict["test"] = test

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.conversion_credit import ConversionCredit # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        matched = ConversionResultMatched(d.pop("matched"))




        _credited = d.pop("credited", UNSET)
        credited: ConversionCredit | Unset
        if isinstance(_credited,  Unset):
            credited = UNSET
        else:
            credited = ConversionCredit.from_dict(_credited)




        test = d.pop("test", UNSET)

        conversion_result = cls(
            id=id,
            matched=matched,
            credited=credited,
            test=test,
        )

        return conversion_result

