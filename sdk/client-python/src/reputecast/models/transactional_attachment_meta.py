from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="TransactionalAttachmentMeta")



@_attrs_define
class TransactionalAttachmentMeta:
    """ 
        Attributes:
            content_type (str):
            filename (str):
            size (int):
            content_id (str | Unset):
     """

    content_type: str
    filename: str
    size: int
    content_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        content_type = self.content_type

        filename = self.filename

        size = self.size

        content_id = self.content_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contentType": content_type,
            "filename": filename,
            "size": size,
        })
        if content_id is not UNSET:
            field_dict["contentId"] = content_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        content_type = d.pop("contentType")

        filename = d.pop("filename")

        size = d.pop("size")

        content_id = d.pop("contentId", UNSET)

        transactional_attachment_meta = cls(
            content_type=content_type,
            filename=filename,
            size=size,
            content_id=content_id,
        )

        return transactional_attachment_meta

