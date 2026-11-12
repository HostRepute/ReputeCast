from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_make_optional import TransactionalMakeOptional





T = TypeVar("T", bound="TransactionalPublishPreviewInput")



@_attrs_define
class TransactionalPublishPreviewInput:
    """ 
        Attributes:
            make_optional (list[TransactionalMakeOptional] | Unset):
     """

    make_optional: list[TransactionalMakeOptional] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_make_optional import TransactionalMakeOptional # noqa: PLC0415
        make_optional: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.make_optional, Unset):
            make_optional = []
            for make_optional_item_data in self.make_optional:
                make_optional_item = make_optional_item_data.to_dict()
                make_optional.append(make_optional_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if make_optional is not UNSET:
            field_dict["makeOptional"] = make_optional

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_make_optional import TransactionalMakeOptional # noqa: PLC0415
        d = dict(src_dict)
        _make_optional = d.pop("makeOptional", UNSET)
        make_optional: list[TransactionalMakeOptional] | Unset = UNSET
        if _make_optional is not UNSET:
            make_optional = []
            for make_optional_item_data in _make_optional:
                make_optional_item = TransactionalMakeOptional.from_dict(make_optional_item_data)



                make_optional.append(make_optional_item)


        transactional_publish_preview_input = cls(
            make_optional=make_optional,
        )

        return transactional_publish_preview_input

