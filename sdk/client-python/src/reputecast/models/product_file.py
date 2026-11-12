from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="ProductFile")



@_attrs_define
class ProductFile:
    """ 
        Attributes:
            content_type (str):
            id (str):
            kind (str): What the content is: pdf, zip, xlsx…, empty when it couldn't be told
            kind_label (str): PDF, ZIP, XLSX
            name (str):
            pages (int):
            size_bytes (int):
            uploaded_at (datetime.datetime):
            uploaded_by (str):
     """

    content_type: str
    id: str
    kind: str
    kind_label: str
    name: str
    pages: int
    size_bytes: int
    uploaded_at: datetime.datetime
    uploaded_by: str





    def to_dict(self) -> dict[str, Any]:
        content_type = self.content_type

        id = self.id

        kind = self.kind

        kind_label = self.kind_label

        name = self.name

        pages = self.pages

        size_bytes = self.size_bytes

        uploaded_at = self.uploaded_at.isoformat()

        uploaded_by = self.uploaded_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contentType": content_type,
            "id": id,
            "kind": kind,
            "kindLabel": kind_label,
            "name": name,
            "pages": pages,
            "sizeBytes": size_bytes,
            "uploadedAt": uploaded_at,
            "uploadedBy": uploaded_by,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        content_type = d.pop("contentType")

        id = d.pop("id")

        kind = d.pop("kind")

        kind_label = d.pop("kindLabel")

        name = d.pop("name")

        pages = d.pop("pages")

        size_bytes = d.pop("sizeBytes")

        uploaded_at = datetime.datetime.fromisoformat(d.pop("uploadedAt"))




        uploaded_by = d.pop("uploadedBy")

        product_file = cls(
            content_type=content_type,
            id=id,
            kind=kind,
            kind_label=kind_label,
            name=name,
            pages=pages,
            size_bytes=size_bytes,
            uploaded_at=uploaded_at,
            uploaded_by=uploaded_by,
        )

        return product_file

