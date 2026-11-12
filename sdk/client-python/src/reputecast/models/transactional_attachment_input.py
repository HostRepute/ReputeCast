from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="TransactionalAttachmentInput")



@_attrs_define
class TransactionalAttachmentInput:
    """ 
        Attributes:
            content (str): The file, base64-encoded
            filename (str):  Example: INV-2026-0418.pdf.
            content_id (str | Unset): Makes it an inline image the HTML references as cid:<content_id>
            content_type (str | Unset): Guessed from the file name when omitted Example: application/pdf.
     """

    content: str
    filename: str
    content_id: str | Unset = UNSET
    content_type: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        content = self.content

        filename = self.filename

        content_id = self.content_id

        content_type = self.content_type


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "content": content,
            "filename": filename,
        })
        if content_id is not UNSET:
            field_dict["content_id"] = content_id
        if content_type is not UNSET:
            field_dict["content_type"] = content_type

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        content = d.pop("content")

        filename = d.pop("filename")

        content_id = d.pop("content_id", UNSET)

        content_type = d.pop("content_type", UNSET)

        transactional_attachment_input = cls(
            content=content,
            filename=filename,
            content_id=content_id,
            content_type=content_type,
        )

        return transactional_attachment_input

