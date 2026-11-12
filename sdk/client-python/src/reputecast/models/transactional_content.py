from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.transactional_header_pair import TransactionalHeaderPair





T = TypeVar("T", bound="TransactionalContent")



@_attrs_define
class TransactionalContent:
    """ 
        Attributes:
            expires_at (datetime.datetime):
            headers (list[TransactionalHeaderPair]):
            html (str):
            masked (list[str]):
            text (str):
     """

    expires_at: datetime.datetime
    headers: list[TransactionalHeaderPair]
    html: str
    masked: list[str]
    text: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_header_pair import TransactionalHeaderPair # noqa: PLC0415
        expires_at = self.expires_at.isoformat()

        headers = []
        for headers_item_data in self.headers:
            headers_item = headers_item_data.to_dict()
            headers.append(headers_item)



        html = self.html

        masked = self.masked



        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "expiresAt": expires_at,
            "headers": headers,
            "html": html,
            "masked": masked,
            "text": text,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_header_pair import TransactionalHeaderPair # noqa: PLC0415
        d = dict(src_dict)
        expires_at = datetime.datetime.fromisoformat(d.pop("expiresAt"))




        headers = []
        _headers = d.pop("headers")
        for headers_item_data in (_headers):
            headers_item = TransactionalHeaderPair.from_dict(headers_item_data)



            headers.append(headers_item)


        html = d.pop("html")

        masked = cast(list[str], d.pop("masked"))


        text = d.pop("text")

        transactional_content = cls(
            expires_at=expires_at,
            headers=headers,
            html=html,
            masked=masked,
            text=text,
        )

        return transactional_content

