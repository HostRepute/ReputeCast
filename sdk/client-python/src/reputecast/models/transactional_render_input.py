from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_address import TransactionalAddress
  from ..models.transactional_render_input_data import TransactionalRenderInputData





T = TypeVar("T", bound="TransactionalRenderInput")



@_attrs_define
class TransactionalRenderInput:
    """ 
        Attributes:
            data (TransactionalRenderInputData | Unset): The template's variables
            language (str | Unset): The language version to render; the linked contact's language, then the fallback, when
                omitted Example: es.
            to (TransactionalAddress | Unset):
     """

    data: TransactionalRenderInputData | Unset = UNSET
    language: str | Unset = UNSET
    to: TransactionalAddress | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_address import TransactionalAddress # noqa: PLC0415
        from ..models.transactional_render_input_data import TransactionalRenderInputData # noqa: PLC0415
        data: dict[str, Any] | Unset = UNSET
        if not isinstance(self.data, Unset):
            data = self.data.to_dict()

        language = self.language

        to: dict[str, Any] | Unset = UNSET
        if not isinstance(self.to, Unset):
            to = self.to.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if data is not UNSET:
            field_dict["data"] = data
        if language is not UNSET:
            field_dict["language"] = language
        if to is not UNSET:
            field_dict["to"] = to

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_address import TransactionalAddress # noqa: PLC0415
        from ..models.transactional_render_input_data import TransactionalRenderInputData # noqa: PLC0415
        d = dict(src_dict)
        _data = d.pop("data", UNSET)
        data: TransactionalRenderInputData | Unset
        if isinstance(_data,  Unset):
            data = UNSET
        else:
            data = TransactionalRenderInputData.from_dict(_data)




        language = d.pop("language", UNSET)

        _to = d.pop("to", UNSET)
        to: TransactionalAddress | Unset
        if isinstance(_to,  Unset):
            to = UNSET
        else:
            to = TransactionalAddress.from_dict(_to)




        transactional_render_input = cls(
            data=data,
            language=language,
            to=to,
        )

        return transactional_render_input

