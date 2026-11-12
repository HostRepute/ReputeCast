from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_test_data_data import TransactionalTestDataData





T = TypeVar("T", bound="TransactionalTestData")



@_attrs_define
class TransactionalTestData:
    """ 
        Attributes:
            data (TransactionalTestDataData):
            name (str):  Example: New user.
     """

    data: TransactionalTestDataData
    name: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_test_data_data import TransactionalTestDataData # noqa: PLC0415
        data = self.data.to_dict()

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "data": data,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_test_data_data import TransactionalTestDataData # noqa: PLC0415
        d = dict(src_dict)
        data = TransactionalTestDataData.from_dict(d.pop("data"))




        name = d.pop("name")

        transactional_test_data = cls(
            data=data,
            name=name,
        )

        return transactional_test_data

