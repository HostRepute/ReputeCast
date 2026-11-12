from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.earn_import_match import EarnImportMatch





T = TypeVar("T", bound="EarnImportGroup")



@_attrs_define
class EarnImportGroup:
    """ 
        Attributes:
            by_customer (int):
            by_email (int):
            count (int):
            examples (list[EarnImportMatch]):
     """

    by_customer: int
    by_email: int
    count: int
    examples: list[EarnImportMatch]





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_import_match import EarnImportMatch # noqa: PLC0415
        by_customer = self.by_customer

        by_email = self.by_email

        count = self.count

        examples = []
        for examples_item_data in self.examples:
            examples_item = examples_item_data.to_dict()
            examples.append(examples_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "byCustomer": by_customer,
            "byEmail": by_email,
            "count": count,
            "examples": examples,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_import_match import EarnImportMatch # noqa: PLC0415
        d = dict(src_dict)
        by_customer = d.pop("byCustomer")

        by_email = d.pop("byEmail")

        count = d.pop("count")

        examples = []
        _examples = d.pop("examples")
        for examples_item_data in (_examples):
            examples_item = EarnImportMatch.from_dict(examples_item_data)



            examples.append(examples_item)


        earn_import_group = cls(
            by_customer=by_customer,
            by_email=by_email,
            count=count,
            examples=examples,
        )

        return earn_import_group

