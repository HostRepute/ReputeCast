from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_template import TransactionalTemplate





T = TypeVar("T", bound="TransactionalPublished")



@_attrs_define
class TransactionalPublished:
    """ 
        Attributes:
            template (TransactionalTemplate):
            version (int):
            previous (int | Unset): The version that was live before (Undo publishes it again)
     """

    template: TransactionalTemplate
    version: int
    previous: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_template import TransactionalTemplate # noqa: PLC0415
        template = self.template.to_dict()

        version = self.version

        previous = self.previous


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "template": template,
            "version": version,
        })
        if previous is not UNSET:
            field_dict["previous"] = previous

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_template import TransactionalTemplate # noqa: PLC0415
        d = dict(src_dict)
        template = TransactionalTemplate.from_dict(d.pop("template"))




        version = d.pop("version")

        previous = d.pop("previous", UNSET)

        transactional_published = cls(
            template=template,
            version=version,
            previous=previous,
        )

        return transactional_published

